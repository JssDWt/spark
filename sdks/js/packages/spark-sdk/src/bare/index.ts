import "fast-text-encoding";

import { setCrypto } from "../utils/crypto.js";
import {
  setFetch,
  SparkFetch,
  SparkHeadersConstructor,
} from "../utils/fetch.js";
import {
  AbortController,
  abortableFetch,
} from "abortcontroller-polyfill/dist/cjs-ponyfill.js";
import { webcrypto } from "bare-crypto";
import bareFetch from "bare-fetch";
import { default as BareHeaders } from "bare-fetch/headers";
import { setAbortController } from "../utils/abortController.js";

const Headers = BareHeaders as SparkHeadersConstructor;

declare const Bare: any;

Bare.on("uncaughtException", (err) => {
  console.error("[Bare] Uncaught Exception:", err);
});

Bare.on("unhandledRejection", (reason, promise) => {
  console.error("[Bare] Unhandled Rejection at:", promise, "reason:", reason);
});

Bare.on("beforeExit", (code) => {
  console.log("[Bare] Process is about to exit with code:", code);
});

function isBufferLike(value: unknown): value is Buffer | Uint8Array {
  return (
    value instanceof Uint8Array ||
    (typeof Buffer !== "undefined" && Buffer.isBuffer(value))
  );
}

const { fetch: abortableBareFetch } = abortableFetch(bareFetch);
const fetchWithSafeBody: SparkFetch = async (input, init = {}) => {
  if (!init.headers) {
    init.headers = new Headers();
  }

  /* bare-fetch does not automatically set Content-Length for body, so we to do it manually to
     prevent bare-http1 from enabling chunk mode which causes a client error later in bare-http1 */
  if (isBufferLike(init.body) && !init.headers.has("Content-Length")) {
    init.headers.set("Content-Length", init.body.byteLength.toString());
  }

  console.log("tmp fetchWithSafeBody await abortableBareFetch(input, init)");
  const result = await abortableBareFetch(input, init);
  console.log("tmp fetchWithSafeBody result:", result);
  return result;
};

setCrypto(webcrypto);
setFetch(fetchWithSafeBody, Headers);
setAbortController(AbortController);

export * from "../errors/index.js";
export { BareSparkSigner } from "../signer/bare-signer.js";
export { SparkWallet } from "../spark-wallet/spark-wallet.js";
export { getLatestDepositTxId } from "../utils/mempool.js";
export * from "../utils/index.js";
