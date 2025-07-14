// Example Bare script using Spark SDK and Frost addon
import { SparkWallet } from '@buildonspark/spark-sdk/bare';
import frost from '@buildonspark/spark-frost-bare-addon';

console.log('Addon hello:', frost.hello());

const wallet = new SparkWallet();
console.log('Created SparkWallet instance', wallet);