import { bytesToHex, hexToBytes } from "@noble/hashes/utils";
import { ValidationError } from "../errors/index.js";
import { createDummyTx, signFrost } from "@buildonspark/spark-frost-bare-addon";
import { IKeyPackage } from "../spark_bindings/types.js";
import { DefaultSparkSigner } from "./signer.js";
import { SignFrostParams, AggregateFrostParams } from "./types.js";

export class BareSparkSigner extends DefaultSparkSigner {
  async signFrost({
    message,
    publicKey,
    verifyingKey,
    selfCommitment,
    statechainCommitments,
    keyDerivation,
    adaptorPubKey,
  }: SignFrostParams): Promise<Uint8Array> {
    const signingPrivateKey =
      await this.getSigningPrivateKeyFromDerivation(keyDerivation);

    if (!signingPrivateKey) {
      throw new ValidationError("Private key not found for public key", {
        field: "privateKey",
      });
    }

    const commitment = selfCommitment.commitment;
    const nonce = this.commitmentToNonceMap.get(commitment);
    if (!nonce) {
      throw new ValidationError("Nonce not found for commitment", {
        field: "nonce",
      });
    }

    const keyPackage: IKeyPackage = {
      secretKey: signingPrivateKey,
      publicKey: publicKey,
      verifyingKey: verifyingKey,
    };

    const statechainCommitmentsArr = statechainCommitments ? Object.entries(statechainCommitments) : [];

    return signFrost(
      message,
      keyPackage,
      nonce,
      selfCommitment,
      statechainCommitmentsArr,
      adaptorPubKey,
    );
  }

  async aggregateFrost({
    message,
    publicKey,
    verifyingKey,
    selfCommitment,
    statechainCommitments,
    adaptorPubKey,
    selfSignature,
    statechainSignatures,
    statechainPublicKeys,
  }: AggregateFrostParams): Promise<Uint8Array> {
    // return NativeSparkFrost.aggregateFrost({
    //   message,
    //   statechainSignatures,
    //   statechainPublicKeys,
    //   verifyingKey,
    //   statechainCommitments,
    //   selfCommitment,
    //   selfPublicKey: publicKey,
    //   selfSignature,
    //   adaptorPubKey,
    // });
    return new Uint8Array([]);
  }
}
