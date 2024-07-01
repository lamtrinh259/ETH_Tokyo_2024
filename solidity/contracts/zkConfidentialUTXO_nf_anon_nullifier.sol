// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import {Groth16Verifier_NF_Anonymity_Nullifier} from "./lib/verifier_nf_anon_nullifier.sol";
import {zkConfidentialUTXONullifier} from "./lib/zkConfidentialUTXONullifier.sol";
import {Registry} from "./lib/registry.sol";
import {Commonlib} from "./lib/common.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {SmtLib} from "@iden3/contracts/lib/SmtLib.sol";
import {PoseidonUnit3L} from "@iden3/contracts/lib/Poseidon.sol";
import "hardhat/console.sol";

uint256 constant MAX_SMT_DEPTH = 64;

/// @title A sample on-chain implementation of a ZKP based C-UTXO pattern with confidentiality, anonymity and history masking
///        The proof has the following statements:
///        - each value in the output commitments must be a positive number in the range 0 ~ (2\*\*40 - 1)
///        - the sum of the nullified values match the sum of output values
///        - the hashes in the input and output match the hash(value, salt, owner public key) formula
///        - the sender possesses the private BabyJubjub key, whose public key is part of the pre-image of the input commitment hashes, which match the corresponding nullifiers
///        - the nullifiers represent input commitments that are included in a Sparse Merkle Tree represented by the root hash
/// @author Kaleido, Inc.
/// @dev Implements double-spend protection with zkp
contract zkConfidentialUTXO_NF_Anonymity_Nullifier is
    zkConfidentialUTXONullifier
{
    Groth16Verifier_NF_Anonymity_Nullifier verifier;

    constructor(
        Groth16Verifier_NF_Anonymity_Nullifier _verifier,
        Registry _registry
    ) zkConfidentialUTXONullifier(_registry) {
        verifier = _verifier;
    }

    /**
     * @dev the main function of the contract.
     *
     * @param nullifier Array of zero or more outputs of a previous `branch()` function call against this
     *      contract that have not yet been spent, and the owner is authorized to spend.
     * @param output Array of zero or more new outputs to generate, for future transactions to spend.
     * @param proof A zero knowledge proof that the submitter is authorized to spend the inputs, and
     *      that the outputs are valid in terms of obeying mass conservation rules.
     *
     * Emits a {UTXOBranch} event.
     */
    function branch(
        uint256 nullifier,
        uint256 output,
        uint256 root,
        Commonlib.Proof calldata proof
    ) public returns (bool) {
        require(
            validateTransactionProposal(
                [nullifier, 0],
                [output, 0],
                root,
                proof
            ),
            "Invalid transaction proposal"
        );

        // construct the public inputs
        uint256[3] memory publicInputs;
        publicInputs[0] = nullifier;
        publicInputs[1] = root;
        publicInputs[2] = output;

        // // Check the proof
        require(
            verifier.verifyProof(proof.pA, proof.pB, proof.pC, publicInputs),
            "Invalid proof"
        );

        processInputsAndOutputs([nullifier, 0], [output, 0]);

        uint256[] memory nullifierArray = new uint256[](1);
        uint256[] memory outputArray = new uint256[](1);
        nullifierArray[0] = nullifier;
        outputArray[0] = output;

        emit UTXOBranch(nullifierArray, outputArray, msg.sender);
        return true;
    }
}