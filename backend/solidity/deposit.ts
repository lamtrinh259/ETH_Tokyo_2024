import {AddressLike, BigNumberish, Contract, ethers, ZeroAddress} from "ethers";
// import {BlindPayment} from "./main";
import {loadProvingKeys, newUser, newUTXO, prepareDepositProof, User, UTXO} from "./util";
// import {SampleEcr20Abi} from "./abi/SampleEcr20Abi"
// import {ZetoAnonAbi} from "./abi/ZetoAnonAbi";
import {useWriteContract} from "wagmi";
import {encodeProof, loadCircuit, Poseidon} from "zeto-js";
import {groth16} from "snarkjs";
import {formatPrivKeyForBabyJub, stringifyBigInts} from "maci-crypto";
import {SampleEcr20Abi} from "./abi/SampleEcr20Abi";
import {ZetoAnonAbi} from "./abi/ZetoAnonAbi";
import { Poseidon, newSalt, tokenUriHash } from "zeto-js";

const ZERO_PUBKEY = [0, 0];


async function prepareProof(circuit: any, provingKey: any, signer: User, inputs: UTXO[], outputs: UTXO[], owners: User[]) {
    const inputCommitments: [BigNumberish, BigNumberish] = inputs.map((input) => input.hash) as [BigNumberish, BigNumberish];
    const inputValues = inputs.map((input) => BigInt(input.value || 0n));
    const inputSalts = inputs.map((input) => input.salt || 0n);
    const outputCommitments: [BigNumberish, BigNumberish] = outputs.map((output) => output.hash) as [BigNumberish, BigNumberish];
    const outputValues = outputs.map((output) => BigInt(output.value || 0n));
    const outputSalts = outputs.map(o => o.salt || 0n);
    const outputOwnerPublicKeys: [[BigNumberish, BigNumberish], [BigNumberish, BigNumberish]] = owners.map(owner => owner.babyJubPublicKey || ZERO_PUBKEY) as [[BigNumberish, BigNumberish], [BigNumberish, BigNumberish]];
    const otherInputs = stringifyBigInts({
        inputOwnerPrivateKey: formatPrivKeyForBabyJub(signer.babyJubPrivateKey),
    });

    const startWitnessCalculation = Date.now();
    const witness = await circuit.calculateWTNSBin(
        {
            inputCommitments,
            inputValues,
            inputSalts,
            outputCommitments,
            outputValues,
            outputSalts,
            outputOwnerPublicKeys,
            ...otherInputs
        },
        true
    );
    const timeWitnessCalculation = Date.now() - startWitnessCalculation;

    const startProofGeneration = Date.now();
    const { proof, publicSignals } = await groth16.prove(provingKey, witness) as unknown as { proof: BigNumberish[]; publicSignals: BigNumberish[] };
    const timeProofGeneration = Date.now() - startProofGeneration;
    console.log(`Witness calculation time: ${timeWitnessCalculation}ms, Proof generation time: ${timeProofGeneration}ms`);
    const encodedProof = encodeProof(proof);
    return {
        inputCommitments,
        outputCommitments,
        encodedProof
    };
}


async function sendTx(
    signer: User,
    inputCommitments: [BigNumberish, BigNumberish],
    outputCommitments: [BigNumberish, BigNumberish],
    outputOwnerAddresses: [AddressLike, AddressLike],
    encodedProof: any,
    zeto:Contract
) {
    const signerAddress = await signer.signer.getAddress();
    const tx = await zeto.transfer(inputCommitments, outputCommitments, encodedProof);
    const results = await tx.wait();
    console.log(`Method transfer() complete. Gas used: ${results?.gasUsed}`);

    return results;
}


async function doTransfer(signer: User, inputs: UTXO[], outputs: UTXO[], owners: User[], zeto:Contract) {
    let inputCommitments: [BigNumberish, BigNumberish];
    let outputCommitments: [BigNumberish, BigNumberish];
    let outputOwnerAddresses: [AddressLike, AddressLike];
    let encodedProof: any;
    const circuit = await loadCircuit('anon');
    const { provingKeyFile: provingKey } = loadProvingKeys('anon');

    const result = await prepareProof(circuit, provingKey, signer, inputs, outputs, owners);
    inputCommitments = result.inputCommitments;
    outputCommitments = result.outputCommitments;
    outputOwnerAddresses = owners.map(owner => owner.ethAddress || ZeroAddress) as [AddressLike, AddressLike];
    encodedProof = result.encodedProof;

    return await sendTx(signer, inputCommitments, outputCommitments, outputOwnerAddresses, encodedProof, zeto);
}

export interface BlindPayment {
    sender: string;
    receiver: string;
}

export const deposit = async (
    {sender,receiver}:BlindPayment
) => {

    const provider = new ethers.JsonRpcProvider("https://sepolia-rpc.scroll.io")

    const signer = new ethers.Wallet(sender,provider);
    const signerUser = await newUser(signer)

    const signer2 = new ethers.Wallet(receiver,provider);
    const signer2User = await newUser(signer2)

    const ecr20 = new ethers.Contract(SampleEcr20Abi.contractAddress, SampleEcr20Abi.abi,signer);
    const zeto = new ethers.Contract(ZetoAnonAbi.contractAddress, ZetoAnonAbi.abi,signer);
    const amount = 100

    // const mintTx = await ecr20.mint(await signer.getAddress(),amount)
    // const approveTx = await ecr20.approve(await zeto.getAddress(),amount)
    //
    // console.log("mintTx",mintTx)
    // console.log("approveTx",approveTx)

    const inputUtx = newUTXO(1, signerUser);
    const inputUtx2 = newUTXO(0, signerUser);

    // const {
    //     outputCommitments,
    //     encodedProof
    // } = await prepareDepositProof(signerUser, inputUtx);


    // const depositTx = await zeto.deposit(amount,outputCommitments[0],encodedProof)
    // console.log(depositTx)



    const outputUtx = newUTXO(1, signer2User);
    const outputUtx2 = newUTXO(0, signerUser, outputUtx.salt);

    // Alice transfers UTXOs to Bob
    const result = await doTransfer(signerUser, [inputUtx,inputUtx2
    ], [outputUtx, outputUtx2], [signerUser, signer2User], zeto);

    console.log(result)
    //
    // const jsonString = JSON.stringify(result, (_, value) =>
    //     typeof value === "bigint" ? value.toString() : value
    // );

}