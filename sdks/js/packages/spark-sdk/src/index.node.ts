/* Root Node.js entrypoint */

import { setIsHermeticTest } from "./tests/isHermeticTest.js";
import { setCrypto } from "./utils/crypto.js";
import nodeCrypto from "crypto";
import fs from "fs";

const cryptoImpl =
  typeof global !== "undefined" && global.crypto ? global.crypto : nodeCrypto;

setCrypto(cryptoImpl);

const isHermeticTest =
  fs.existsSync("/tmp/spark_hermetic") || process.env.HERMETIC_TEST === "true";

setIsHermeticTest(isHermeticTest);

export * from "./errors/index.js";
export * from "./utils/index.js";

export {
  DefaultSparkSigner,
  TaprootSparkSigner,
  TaprootOutputKeysGenerator,
  type SparkSigner,
  type TokenSigner,
} from "./signer/signer.js";
export * from "./signer/types.js";

export { SparkWallet } from "./spark-wallet/spark-wallet.js";
export * from "./spark-wallet/types.js";

export { WalletConfig } from "./services/wallet-config.js";
export { TokenTransactionService } from "./services/token-transactions.js";
export { type ConnectionManager } from "./services/connection.js";
export { type WalletConfigService } from "./services/config.js";
export { type ConfigOptions } from "./services/wallet-config.js";
