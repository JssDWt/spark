// const NODE_ID = "LightsparkNodeWithOSKLND:018d28e7-dc4a-f96b-0000-f276772e6fb1";
// const NODE_PASSWORD = "1234!@#$";
// const CLIENT_ID = "0196646ed4004f890000d583540a2aa4";
// const CLIENT_SECRET = "jl-tlTukryTnVR--DRD2NEvvGg2Biz5taMecGtRGJIU";
// describe("Spark SDK init", () => {
//   test("should initialize a new wallet and return a mnemonic", async () => {
//     const { wallet, mnemonic } = await SparkWallet.initialize({
//       options: {
//         network: "REGTEST",
//       },
//     });
//     expect(wallet).toBeDefined();
//     expect(mnemonic?.length).toBeGreaterThan(0);
//   });

//   test("should init an existing wallet from mnemonic, return the mnemonic and balance is 10000 sats", async () => {
//     const { wallet, mnemonic } = await SparkWallet.initialize({
//       mnemonicOrSeed:
//         "hour cattle reason lamp hen slice square heart pigeon fat print upgrade",
//       options: {
//         network: "REGTEST",
//       },
//     });
//     const balance = await wallet.getBalance();
//     expect(mnemonic).toBe(
//       "hour cattle reason lamp hen slice square heart pigeon fat print upgrade",
//     );
//     expect(balance.balance).toBe(10000n);
//   });
// });

// describe("Spark Deposit tests", () => {
//   let wallet: SparkWallet;
//   let browser: Browser;

//   beforeAll(async () => {
//     ({ wallet } = await SparkWallet.initialize({
//       options: {
//         network: "REGTEST",
//       },
//     }));
//     browser = await puppeteer.launch({
//       headless: HEADLESS,
//       defaultViewport: { width: 1280, height: 800 },
//     });
//   });

//   afterAll(async () => {
//     await browser.close();
//   });

//   test("should generate a L1 address", async () => {
//     const address = await wallet.getSingleUseDepositAddress();
//     expect(address.length).toBeGreaterThan(0);
//   });

//   test("Spark SDK L1 deposit should occur within 60 seconds", async () => {
//     // TODO: lower from 60 seconds when deposits get faster
//     const address = await wallet.getSingleUseDepositAddress();
//     expect(address.length).toBeGreaterThan(0);

//     // Deposit using test faucet
//     const page = await loadFaucetPage(browser);

//     // Populate fields and send mock L1 deposit
//     await depositFunds(page, address);

//     // Check for balance update
//     let walletBalance = await getDepositBalance(address, wallet);
//     expect(walletBalance?.balance).toBe(10000n);
//   }, 60000);

//   test("should not update balance from additional payments to the same address", async () => {
//     const address = await wallet.getSingleUseDepositAddress();
//     expect(address.length).toBeGreaterThan(0);

//     // Deposit using test faucet
//     const page = await loadFaucetPage(browser);
//     // Populate fields and send mock L1 deposit
//     await depositFunds(page, address);
//     // Check for balance update
//     let walletBalance = await getDepositBalance(address, wallet);
//     expect(walletBalance?.balance).toBe(20000n);

//     await depositFunds(page, address);
//     walletBalance = await getDepositBalance(address, wallet);
//     expect(walletBalance?.balance).toBe(20000n);

//     await depositFunds(page, address);
//     walletBalance = await getDepositBalance(address, wallet);
//     expect(walletBalance?.balance).toBe(20000n);
//   }, 60000);

//   test("should update balance with payments to new L1 addresses", async () => {
//     let address = await wallet.getSingleUseDepositAddress();
//     expect(address.length).toBeGreaterThan(0);

//     // Deposit using test faucet
//     const page = await loadFaucetPage(browser);
//     // Populate fields and send mock L1 deposit
//     await depositFunds(page, address);
//     // Check for balance update
//     let walletBalance = await getDepositBalance(address, wallet);
//     expect(walletBalance?.balance).toBe(30000n);

//     address = await wallet.getSingleUseDepositAddress();
//     expect(address.length).toBeGreaterThan(0);
//     await depositFunds(page, address);
//     walletBalance = await getDepositBalance(address, wallet);
//     expect(walletBalance?.balance).toBe(40000n);

//     address = await wallet.getSingleUseDepositAddress();
//     expect(address.length).toBeGreaterThan(0);
//     await depositFunds(page, address);
//     walletBalance = await getDepositBalance(address, wallet);
//     expect(walletBalance?.balance).toBe(50000n);
//   }, 60000);
// });

// describe("Spark transactions", () => {
//   let wallet1: SparkWallet, wallet2: SparkWallet;
//   let browser: Browser;

//   beforeAll(async () => {
//     ({ wallet: wallet1 } = await SparkWallet.initialize({
//       options: {
//         network: "REGTEST",
//       },
//     }));

//     ({ wallet: wallet2 } = await SparkWallet.initialize({
//       options: {
//         network: "REGTEST",
//       },
//     }));

//     browser = await puppeteer.launch({
//       headless: HEADLESS,
//       defaultViewport: { width: 1280, height: 800 },
//     });

//     let address = await wallet1.getSingleUseDepositAddress();
//     const page = await loadFaucetPage(browser);
//     await depositFunds(page, address);
//     let balance = await getDepositBalance(address, wallet1);
//     expect(balance?.balance).toBe(10000n);
//   }, 60000);

//   afterAll(async () => {
//     await browser.close();
//   });

//   test("should generate Spark address", async () => {
//     const address = await wallet1.getSparkAddress();
//     expect(address.length).toBeGreaterThan(0);
//   });

//   test("should send 1 sat from wallet 1 to wallet 2 and check balances in 10s", async () => {
//     const address = await wallet2.getSparkAddress();
//     await wallet1.transfer({
//       receiverSparkAddress: address,
//       amountSats: 1,
//     });
//     await wallet2.claimTransfers();
//     const wallet1Balance = await wallet1.getBalance();
//     const wallet2Balance = await wallet2.getBalance();
//     expect(wallet1Balance.balance).toBe(9999n);
//     expect(wallet2Balance.balance).toBe(1n);
//   }, 10000);

//   test("should send 9999 sats from wallet 1 to wallet 2 and check balances in 10s", async () => {
//     const address = await wallet2.getSparkAddress();
//     await wallet1.transfer({
//       receiverSparkAddress: address,
//       amountSats: 9999,
//     });
//     await wallet2.claimTransfers();
//     const wallet1Balance = await wallet1.getBalance();
//     const wallet2Balance = await wallet2.getBalance();
//     expect(wallet1Balance.balance).toBe(0n);
//     expect(wallet2Balance.balance).toBe(10000n);
//   }, 10000);

//   test("should throw error when sending more than wallet balance", async () => {
//     const address = await wallet2.getSparkAddress();
//     await expect(
//       wallet1.transfer({
//         receiverSparkAddress: address,
//         amountSats: 1,
//       }),
//     ).rejects.toThrow();

//     await expect(
//       wallet1.transfer({
//         receiverSparkAddress: address,
//         amountSats: 1000,
//       }),
//     ).rejects.toThrow();
//   }, 10000);

//   test("should be able send 2 payments to the same spark address and update balance accordingly", async () => {
//     const address = await wallet1.getSparkAddress();
//     await wallet2.transfer({
//       receiverSparkAddress: address,
//       amountSats: 5000,
//     });
//     await wallet1.claimTransfers();
//     let wallet1Balance = await wallet1.getBalance();
//     let wallet2Balance = await wallet2.getBalance();
//     expect(wallet1Balance.balance).toBe(5000n);
//     expect(wallet2Balance.balance).toBe(5000n);
//     await wallet2.transfer({
//       receiverSparkAddress: address,
//       amountSats: 5000,
//     });
//     await wallet1.claimTransfers();
//     wallet1Balance = await wallet1.getBalance();
//     wallet2Balance = await wallet2.getBalance();
//     expect(wallet1Balance.balance).toBe(10000n);
//     expect(wallet2Balance.balance).toBe(0n);
//   }, 10000);

//   test("should throw error when sending from REGTEST to MAINNET", async () => {
//     const { wallet: mainnetWallet } = await SparkWallet.initialize({
//       options: {
//         network: "MAINNET",
//       },
//     });

//     const address = await mainnetWallet.getSparkAddress();
//     await expect(
//       wallet1.transfer({
//         receiverSparkAddress: address,
//         amountSats: 5000,
//       }),
//     ).rejects.toThrow();
//   }, 10000);

//   test.concurrent.each(Array.from({ length: 10 }, (_, i) => [i]))(
//     "should successfully complete back and forth transaction %i between wallets",
//     async () => {
//       let wallet1Balance = await wallet1.getBalance();
//       let wallet2Balance = await wallet2.getBalance();

//       // First transfer: wallet1 -> wallet2
//       const wallet2SparkAddress = await wallet2.getSparkAddress();
//       const transferAmount1 = Number(
//         (wallet1Balance.balance * BigInt(Math.floor(Math.random() * 101))) /
//           BigInt(100),
//       );

//       await wallet1.transfer({
//         receiverSparkAddress: wallet2SparkAddress,
//         amountSats: transferAmount1,
//       });
//       await wallet2.claimTransfers();

//       let updatedWallet1Balance = await wallet1.getBalance();
//       let updatedWallet2Balance = await wallet2.getBalance();

//       expect(wallet1Balance.balance - BigInt(transferAmount1)).toBe(
//         updatedWallet1Balance.balance,
//       );
//       expect(wallet2Balance.balance + BigInt(transferAmount1)).toBe(
//         updatedWallet2Balance.balance,
//       );

//       // Second transfer: wallet2 -> wallet1
//       const wallet1SparkAddress = await wallet1.getSparkAddress();
//       const transferAmount2 = Number(
//         (updatedWallet2Balance.balance *
//           BigInt(Math.floor(Math.random() * 101))) /
//           BigInt(100),
//       );

//       await wallet2.transfer({
//         receiverSparkAddress: wallet1SparkAddress,
//         amountSats: transferAmount2,
//       });
//       await wallet1.claimTransfers();

//       const finalWallet1Balance = await wallet1.getBalance();
//       const finalWallet2Balance = await wallet2.getBalance();

//       expect(updatedWallet1Balance.balance + BigInt(transferAmount2)).toBe(
//         finalWallet1Balance.balance,
//       );
//       expect(updatedWallet2Balance.balance - BigInt(transferAmount2)).toBe(
//         finalWallet2Balance.balance,
//       );
//     },
//     120000,
//   );
// });

// describe("Lightning transactions", () => {
//   let wallet1: SparkWallet, wallet2: SparkWallet;
//   let browser: Browser;
//   let client: LightsparkClient;

//   beforeAll(async () => {
//     ({ wallet: wallet1 } = await SparkWallet.initialize({
//       options: {
//         network: "REGTEST",
//       },
//     }));
//     ({ wallet: wallet2 } = await SparkWallet.initialize({
//       options: {
//         network: "REGTEST",
//       },
//     }));
//     client = new LightsparkClient(
//       new AccountTokenAuthProvider(CLIENT_ID, CLIENT_SECRET),
//     );

//     // Pass in your signing secret to unlock the node
//     client.loadNodeSigningKey(NODE_ID, {
//       password: NODE_PASSWORD,
//     });
//     browser = await puppeteer.launch({
//       headless: HEADLESS,
//       defaultViewport: { width: 1280, height: 800 },
//     });
//     let address = await wallet1.getSingleUseDepositAddress();
//     const page = await loadFaucetPage(browser);
//     await depositFunds(page, address);
//     let balance = await getDepositBalance(address, wallet1);
//     expect(balance?.balance).toBe(10000n);
//   }, 60000);

//   afterAll(async () => {
//     await browser.close();
//   });

//   test("should generate 1000 sat Lightning invoice", async () => {
//     const invoice = await wallet1.createLightningInvoice({
//       amountSats: 1000,
//       memo: "test",
//     });

//     expect(invoice).toBeDefined();
//     expect(invoice.invoice.encodedInvoice.length).toBeGreaterThan(0);
//     expect(invoice.invoice.amount.originalValue).toEqual(1000000);
//   }, 10000);

//   test("should return Lightning fee estimate with Lightspark connect node", async () => {
//     const invoice = await client.createInvoice(
//       NODE_ID,
//       10000,
//       "fee estimate test",
//     );
//     const feeEstimate = await wallet1.getLightningSendFeeEstimate({
//       encodedInvoice: invoice!,
//     });
//     expect(feeEstimate).toBeDefined();
//     expect(feeEstimate?.feeEstimate.originalValue).toBeGreaterThanOrEqual(0);
//   }, 10000);

//   test("should send lightning payment from wallet 1 to wallet 2", async () => {
//     const amount = 5000;
//     const invoice = await wallet2.createLightningInvoice({
//       amountSats: amount,
//       memo: "test",
//     });
//     wallet1.payLightningInvoice({ invoice: invoice.invoice.encodedInvoice });

//     // wait for tx to complete
//     while (!(await wallet2.claimTransfers())) {
//       await sleep(3000);
//     }
//     let wallet1Balance = await wallet1.getBalance();
//     let wallet2Balance = await wallet2.getBalance();

//     expect(wallet1Balance.balance).toBe(10000n - BigInt(amount));
//     expect(wallet2Balance.balance).toBe(BigInt(amount));
//   }, 10000);

//   test("should send lightning payment from wallet 1 to LS connect node", async () => {
//     const amount = 1000;
//     const invoice = await client.createInvoice(NODE_ID, amount, "node test");
//     let node = await client.executeRawQuery(getNodeQuery(NODE_ID));
//     const wallet1StartingBalance = await wallet1.getBalance();
//     const startingBalance = (node as LightsparkNodeWithOSK)?.balances
//       ?.availableToSendBalance.originalValue;
//     let currBalance = startingBalance;
//     await wallet1.payLightningInvoice({ invoice: invoice! });
//     while (currBalance == startingBalance) {
//       node = await client.executeRawQuery(getNodeQuery(NODE_ID));
//       currBalance = (node as LightsparkNodeWithOSK)?.balances
//         ?.availableToSendBalance.originalValue;
//     }
//     let wallet1Balance = await wallet1.getBalance();
//     expect(wallet1Balance.balance).toBe(
//       wallet1StartingBalance.balance - BigInt(amount),
//     );
//     expect(currBalance).toBe(BigInt(startingBalance! + amount * 1000));
//   }, 10000);

//   test("should send lightning payment from LS connect node to wallet 1", async () => {
//     const amount = 1000;
//     const invoice = await await wallet1.createLightningInvoice({
//       amountSats: 1000,
//       memo: "test",
//     });
//     const wallet1StartingBalance = await wallet1.getBalance();
//     let node = await client.executeRawQuery(getNodeQuery(NODE_ID));
//     const startingBalance = (node as LightsparkNodeWithOSK)?.balances
//       ?.availableToSendBalance.originalValue;
//     let currBalance = startingBalance;
//     await client.payInvoice(NODE_ID, invoice.invoice.encodedInvoice, 10000);

//     while (currBalance == startingBalance) {
//       node = await client.executeRawQuery(getNodeQuery(NODE_ID));
//       currBalance = (node as LightsparkNodeWithOSK)?.balances
//         ?.availableToSendBalance.originalValue;
//     }
//     let wallet1Balance = await wallet1.getBalance();
//     expect(wallet1Balance.balance).toBe(
//       wallet1StartingBalance.balance + BigInt(amount),
//     );
//     expect(currBalance).toBe(BigInt(startingBalance! - amount * 1000));
//   }, 10000);

//   test("should throw error when paying expired invoice", async () => {
//     const expiredInvoice =
//       "lnbcrt59114920p1pnlmqsvpp5slq5l7y8ly9grpw6h8cd2hu5rc3d0rvs5j3vdecvz2fjlpnvtplqhp57vtwrpzq5qcxvfvadqfce4fpjyvkcafuelwh6af2rr0vrt2et2uqcqzpgxqzfvrzjqgjnjdd9wqax7ppfpq0q35k7lns04g2lf46nq5cz9pr4r4f6fcrqsqqqqz57wrw2m5qqqqqqqqqqqqqq9qsp5un68cnsu95ahfgphs440wq5vfr8uws2ncqr5260lr6vs948j6k9s9qxpqysgqx5xp8cp78jnjrugck0vna5dmyx9kf8djjc4p05uu4xupdajnq60yq92ljt6wg8en2c5757jp5wc5t2ft0dkppk86twmchn839fnalasqg598zz";
//     await expect(() => {
//       wallet1.payLightningInvoice({ invoice: expiredInvoice });
//     }).rejects.toThrow();
//   }, 10000);
// });


// async function depositFunds(
//   page: Page,
//   address: string,
//   amount: string = "10000",
// ) {
//   const amountSelector = "input#satoshis";
//   const addressSelector = "input#address";
//   await page.waitForSelector(amountSelector, {
//     visible: true,
//     timeout: 5000,
//   });

//   await page.click(amountSelector, { clickCount: 3 }); // select all text
//   await page.keyboard.press("Backspace"); // or 'Delete' on macOS
//   await page.type(amountSelector, amount);
//   await page.waitForSelector(addressSelector, { visible: true, timeout: 5000 });
//   await page.click(addressSelector, { clickCount: 3 }); // select all text
//   await page.keyboard.press("Backspace"); // or 'Delete' on macOS
//   await page.type(addressSelector, address);
//   await page
//     .locator(
//       '::-p-xpath(//button[@type="submit" and contains(@class,"submit-button") and normalize-space()="Request Regtest Coins"])',
//     )
//     .click();
// }

// async function loadFaucetPage(browser: Browser) {
//   const page = await browser.newPage();
//   await page.authenticate({
//     username: "hackathon",
//     password: "MakeBitcoinMoneyAgain",
//   });
//   // Navigate to the faucet page
//   await page.goto("https://regtest-mempool.dev.dev.sparkinfra.net/faucet", {
//     waitUntil: "networkidle2",
//   });
//   return page;
// }


// async function getDepositBalance(address: string, wallet: SparkWallet) {
//   let walletBalance;
//   for (let i = 0; i < 20; i++) {
//     const result = await getLatestDepositTxId(address);
//     let tx;
//     if (result) {
//       tx = await wallet.claimDeposit(result);
//     }
//     if (tx) {
//       walletBalance = await wallet.getBalance();
//       break;
//     }
//     await sleep(3000); // Wait 3 seconds before checking again
//   }
//   return walletBalance;
// }
