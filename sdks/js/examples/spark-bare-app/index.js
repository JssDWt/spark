// Example Bare script using Spark SDK and Frost addon
import { SparkWallet } from '@buildonspark/spark-sdk/bare';
import frost from '@buildonspark/spark-frost-bare-addon';

console.log('Addon hello:', frost.hello());

const wallet = new SparkWallet();

try {
  const initWalletResult = await wallet.initWallet();
} catch (error) {
  console.error('Error initializing wallet:', error);
}

console.log('Wallet initialized:', wallet);
