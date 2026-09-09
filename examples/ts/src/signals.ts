import * as path from "node:path";
import * as grpc from "@grpc/grpc-js";
import * as protoLoader from "@grpc/proto-loader";
import bs58 from "bs58";

const PROTO_DIR = path.resolve(__dirname, "../../../proto/deshred-v1");
const TARGET = process.argv[2] ?? "127.0.0.1:9900";

const definition = protoLoader.loadSync(
  ["signals.proto", "intents.proto", "routes.proto"],
  { includeDirs: [PROTO_DIR], longs: String, enums: String, defaults: true },
);

const proto = grpc.loadPackageDefinition(definition) as any;
const client = new proto.deshred.v1.SignalStream(
  TARGET,
  grpc.credentials.createInsecure(),
);

console.log(`subscribing to ${TARGET} deshred.v1.SignalStream/SubscribeTxSignals`);
const stream = client.SubscribeTxSignals({});

stream.on("data", (signal: any) => {
  const signature = bs58.encode(signal.signature);
  const keys = signal.account_keys_packed?.length
    ? signal.account_keys_packed.length / 32
    : 0;
  const fee =
    signal.priority_fee_gap === "PRIORITY_FEE_GAP_NONE"
      ? `${signal.priority_fee_lamports} lamports`
      : `none (${signal.priority_fee_gap})`;
  console.log(
    `slot ${signal.slot} tx ${signal.tx_index} keys ${keys} fee ${fee} ${signature.slice(0, 16)}…`,
  );
});

stream.on("error", (error: grpc.ServiceError) => {
  if (error.code === grpc.status.UNIMPLEMENTED) {
    console.error(`refused: ${error.details}`);
    process.exit(1);
  }
  console.error(error);
  process.exit(1);
});
