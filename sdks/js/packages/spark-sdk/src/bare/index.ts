import "fast-text-encoding";

import { setCrypto } from "../utils/crypto.js";
import { webcrypto } from "bare-crypto";

setCrypto(webcrypto);

export * from "../errors/index.js";
export { BareSparkSigner } from "../signer/bare-signer.js";
export { SparkWallet } from "../spark-wallet/spark-wallet.js";
export { getLatestDepositTxId } from "../utils/mempool.js";
export * from "../utils/index.js";
