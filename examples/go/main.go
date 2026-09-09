package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/mr-tron/base58"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "deshred-entries-go/gen/shredstream"
)

type cursor struct {
	buf []byte
	at  int
}

func (c *cursor) u64() (uint64, error) {
	if c.at+8 > len(c.buf) {
		return 0, fmt.Errorf("u64 at %d exceeds %d bytes", c.at, len(c.buf))
	}
	v := binary.LittleEndian.Uint64(c.buf[c.at:])
	c.at += 8
	return v, nil
}

func (c *cursor) compactU16() (int, error) {
	value, shift := 0, 0
	for {
		if c.at >= len(c.buf) {
			return 0, fmt.Errorf("compact-u16 at %d exceeds %d bytes", c.at, len(c.buf))
		}
		b := c.buf[c.at]
		c.at++
		value |= int(b&0x7f) << shift
		if b < 0x80 {
			return value, nil
		}
		shift += 7
	}
}

func (c *cursor) skip(n int) error {
	if n < 0 || c.at+n > len(c.buf) {
		return fmt.Errorf("skip %d at %d exceeds %d bytes", n, c.at, len(c.buf))
	}
	c.at += n
	return nil
}

func (c *cursor) peek() (byte, error) {
	if c.at >= len(c.buf) {
		return 0, fmt.Errorf("peek at %d exceeds %d bytes", c.at, len(c.buf))
	}
	return c.buf[c.at], nil
}

type transaction struct {
	firstSignature []byte
	version        int
}

// Consumes exactly one wire transaction: signatures, then the message. There is no length prefix
// between transactions in an entry, so the only way to find the next one is to walk this one.
func (c *cursor) transaction() (transaction, error) {
	var tx transaction
	signatures, err := c.compactU16()
	if err != nil {
		return tx, err
	}
	for i := 0; i < signatures; i++ {
		if c.at+64 > len(c.buf) {
			return tx, fmt.Errorf("signature %d truncated", i)
		}
		if i == 0 {
			tx.firstSignature = c.buf[c.at : c.at+64]
		}
		c.at += 64
	}

	tx.version = -1
	prefix, err := c.peek()
	if err != nil {
		return tx, err
	}
	if prefix&0x80 != 0 {
		tx.version = int(prefix & 0x7f)
		c.at++
	}

	if err := c.skip(3); err != nil { // num_required_signatures, ro_signed, ro_unsigned
		return tx, err
	}
	keys, err := c.compactU16()
	if err != nil {
		return tx, err
	}
	if err := c.skip(32 * keys); err != nil {
		return tx, err
	}
	if err := c.skip(32); err != nil { // recent blockhash
		return tx, err
	}

	instructions, err := c.compactU16()
	if err != nil {
		return tx, err
	}
	for i := 0; i < instructions; i++ {
		if err := c.skip(1); err != nil { // program id index
			return tx, err
		}
		accounts, err := c.compactU16()
		if err != nil {
			return tx, err
		}
		if err := c.skip(accounts); err != nil {
			return tx, err
		}
		data, err := c.compactU16()
		if err != nil {
			return tx, err
		}
		if err := c.skip(data); err != nil {
			return tx, err
		}
	}

	if tx.version == 0 {
		lookups, err := c.compactU16()
		if err != nil {
			return tx, err
		}
		for i := 0; i < lookups; i++ {
			if err := c.skip(32); err != nil { // table account
				return tx, err
			}
			writable, err := c.compactU16()
			if err != nil {
				return tx, err
			}
			if err := c.skip(writable); err != nil {
				return tx, err
			}
			readonly, err := c.compactU16()
			if err != nil {
				return tx, err
			}
			if err := c.skip(readonly); err != nil {
				return tx, err
			}
		}
	}
	return tx, nil
}

func decodeEntries(payload []byte) ([]transaction, error) {
	c := &cursor{buf: payload}
	entries, err := c.u64()
	if err != nil {
		return nil, err
	}
	var out []transaction
	for e := uint64(0); e < entries; e++ {
		if _, err := c.u64(); err != nil { // num_hashes
			return nil, err
		}
		if err := c.skip(32); err != nil { // hash
			return nil, err
		}
		count, err := c.u64()
		if err != nil {
			return nil, err
		}
		for t := uint64(0); t < count; t++ {
			tx, err := c.transaction()
			if err != nil {
				return nil, fmt.Errorf("entry %d tx %d: %w", e, t, err)
			}
			out = append(out, tx)
		}
	}
	// The self-check that makes this walk trustworthy: a correct walk lands exactly on the end.
	if c.at != len(payload) {
		return out, fmt.Errorf("walked %d of %d bytes — a transaction boundary was misread", c.at, len(payload))
	}
	return out, nil
}

func main() {
	target := "127.0.0.1:9900"
	if len(os.Args) > 1 {
		target = os.Args[1]
	}
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("dial %s: %v", target, err)
	}
	defer conn.Close()

	stream, err := pb.NewShredstreamProxyClient(conn).SubscribeEntries(
		context.Background(), &pb.SubscribeEntriesRequest{},
	)
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}
	fmt.Printf("subscribing to %s\n", target)

	for {
		frame, err := stream.Recv()
		if err != nil {
			log.Fatalf("recv: %v", err)
		}
		transactions, err := decodeEntries(frame.GetEntries())
		if err != nil {
			fmt.Printf("slot %12d  %d txs before: %v\n", frame.GetSlot(), len(transactions), err)
			continue
		}
		first := "-"
		versions := map[int]int{}
		for i, tx := range transactions {
			if i == 0 && tx.firstSignature != nil {
				first = base58.Encode(tx.firstSignature)
			}
			versions[tx.version]++
		}
		fmt.Printf("slot %12d  txs %4d  legacy %3d  v0 %3d  first %s\n",
			frame.GetSlot(), len(transactions), versions[-1], versions[0], first)
	}
}
