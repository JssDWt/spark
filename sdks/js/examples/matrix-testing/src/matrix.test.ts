import { SparkWallet } from "@buildonspark/spark-sdk";
import fs from "fs";
import path from "path";
import { expect, jest, test } from "@jest/globals";

type SdkVersions = Record<string, any>;

const { wallet: fundingWallet } = await SparkWallet.initialize({
  mnemonicOrSeed:
    "media divide brave fence shrimp seat skate person avocado dumb figure satoshi",
});
const fundingWalletAddress = await fundingWallet.getSparkAddress();

async function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function loadSdkVersions(): Promise<SdkVersions> {
  // 1. Read and parse package.json
  const packageJsonPath = path.join(process.cwd(), "package.json");
  const packageJsonContent = fs.readFileSync(packageJsonPath, "utf8");
  const packageJson = JSON.parse(packageJsonContent);

  // 2. Combine dependencies and devDependencies to search in both
  const allDependencies = {
    ...packageJson.dependencies,
    ...packageJson.devDependencies,
  };

  // 3. Find all keys that match the "spark-sdk-" pattern
  const sdkAliases = Object.keys(allDependencies).filter((key) =>
    key.startsWith("spark-sdk-")
  );

  // 4. Dynamically import each SDK and store it in a map
  const sdkModules: SdkVersions = {};
  console.log(`Found SDK versions to test: ${sdkAliases.join(", ")}`);

  for (const alias of sdkAliases) {
    try {
      // Use dynamic import() which returns a Promise
      const module = await import(alias);

      console.log(`Initializing version: ${alias}`);

      let { wallet, mnemonic } = await module.SparkWallet.initialize({
        options: {
          network: "REGTEST",
        },
      });
      sdkModules[alias] = wallet;
      await sleep(1000);
    } catch (error) {
      console.error(`Failed to import SDK version: ${alias}`, error);
    }
  }
  return sdkModules;
}

let wallets = await loadSdkVersions();

describe.each(Object.entries(wallets))(
  "Sending sdk version %s",
  (version: string, currSender: any) => {
    beforeAll(async () => {
      console.log(`Funding wallet for sdk version: ${version}`);
      const address = await currSender.getSparkAddress();
      const transfer = await fundingWallet.transfer({
        amountSats: 1111,
        receiverSparkAddress: address,
      });
      while ((await currSender.getBalance()).balance == 0) {
        await sleep(1000);
        console.log("waiting");
      }
      console.log(
        `Funding wallet balance: ${(await currSender.getBalance()).balance}`
      );
    }, 30000);

    test.each(Object.entries(wallets))(
      "Receiving sdk version %s - Spark",
      async (version: string, currReceiver: any) => {
        if (currSender !== currReceiver) {
          const address = await currReceiver.getSparkAddress();
          let currBalance: bigint;
          const transfer = await currSender.transfer({
            amountSats: 1,
            receiverSparkAddress: address,
          });
          await sleep(5000);
          currBalance = (await currReceiver.getBalance()).balance;
          console.log(`Current balance: ${currBalance}`);
          expect(currBalance).toBe(1n);

          await currSender.transfer({
            amountSats: 10,
            receiverSparkAddress: address,
          });
          await sleep(5000);
          expect((await currReceiver.getBalance()).balance).toBe(11n);

          await currSender.transfer({
            amountSats: 100,
            receiverSparkAddress: address,
          });
          await sleep(5000);
          expect((await currReceiver.getBalance()).balance).toBe(111n);

          await currSender.transfer({
            amountSats: 1000,
            receiverSparkAddress: address,
          });
          await sleep(5000);
          expect((await currReceiver.getBalance()).balance).toBe(1111n);

          await currReceiver.transfer({
            amountSats: 1111,
            receiverSparkAddress: await currSender.getSparkAddress(),
          });
          await sleep(5000);
          expect((await currReceiver.getBalance()).balance).toBe(0n);
          expect((await currSender.getBalance()).balance).toBe(1111n);
        } else {
          expect(true).toBe(true);
        }
      },
      120000
    );

    afterAll(async () => {
      await currSender.transfer({
        amountSats: 1111,
        receiverSparkAddress: fundingWalletAddress,
      });
    }, 30000);
  }
);
