// Copyright © 2024 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
pragma solidity ^0.8.20;

import {Groth16Verifier_CheckHashesValue} from "./lib/verifier_check_hashes_value.sol";
import {Groth16Verifier_CheckInputsOutputsValue} from "./lib/verifier_check_inputs_outputs_value.sol";
import {Groth16Verifier_Anon} from "./lib/verifier_anon.sol";
import {Registry} from "./lib/registry.sol";
import {Commonlib} from "./lib/common.sol";
import {ZetoBase} from "./lib/zeto_base.sol";
import {ZetoFungible} from "./lib/zeto_fungible.sol";
import {ZetoFungibleWithdraw} from "./lib/zeto_fungible_withdraw.sol";
import {Ownable} from "@openzeppelin/contracts/access/Ownable.sol";
import {UUPSUpgradeable} from "@openzeppelin/contracts-upgradeable/proxy/utils/UUPSUpgradeable.sol";
import "hardhat/console.sol";

/// @title A sample implementation of a Zeto based fungible token with anonymity and no encryption
/// @author Kaleido, Inc.
/// @dev The proof has the following statements:
///        - each value in the output commitments must be a positive number in the range 0 ~ (2\*\*40 - 1)
///        - the sum of the input values match the sum of output values
///        - the hashes in the input and output match the `hash(value, salt, owner public key)` formula
///        - the sender possesses the private BabyJubjub key, whose public key is part of the pre-image of the input commitment hashes
contract Zeto_Anon is ZetoBase, ZetoFungibleWithdraw, UUPSUpgradeable {
    Groth16Verifier_Anon internal verifier;

    function initialize(
        address initialOwner,
        Groth16Verifier_CheckHashesValue _depositVerifier,
        Groth16Verifier_CheckInputsOutputsValue _withdrawVerifier,
        Groth16Verifier_Anon _verifier
    ) public initializer {
        __ZetoBase_init(initialOwner);
        __ZetoFungibleWithdraw_init(_depositVerifier, _withdrawVerifier);
        verifier = _verifier;
    }

    function _authorizeUpgrade(address) internal override onlyOwner {}

    /**
     * @dev the main function of the contract.
     *
     * @param inputs Array of UTXOs to be spent by the transaction.
     * @param outputs Array of new UTXOs to generate, for future transactions to spend.
     * @param proof A zero knowledge proof that the submitter is authorized to spend the inputs, and
     *      that the outputs are valid in terms of obeying mass conservation rules.
     *
     * Emits a {UTXOTransfer} event.
     */
    function transfer(
        uint256[2] memory inputs,
        uint256[2] memory outputs,
        Commonlib.Proof calldata proof
    ) public returns (bool) {
        require(
            validateTransactionProposal(inputs, outputs, proof),
            "Invalid transaction proposal"
        );

        // construct the public inputs
        uint256[4] memory publicInputs;
        publicInputs[0] = inputs[0];
        publicInputs[1] = inputs[1];
        publicInputs[2] = outputs[0];
        publicInputs[3] = outputs[1];

        // Check the proof
        require(
            verifier.verifyProof(proof.pA, proof.pB, proof.pC, publicInputs),
            "Invalid proof"
        );

        processInputsAndOutputs(inputs, outputs);

        uint256[] memory inputArray = new uint256[](inputs.length);
        uint256[] memory outputArray = new uint256[](outputs.length);
        for (uint256 i = 0; i < inputs.length; ++i) {
            inputArray[i] = inputs[i];
            outputArray[i] = outputs[i];
        }
        emit UTXOTransfer(inputArray, outputArray, msg.sender);

        return true;
    }

    function deposit(
        uint256 amount,
        uint256 utxo,
        Commonlib.Proof calldata proof
    ) public {
        _deposit(amount, utxo, proof);
        uint256[] memory utxos = new uint256[](1);
        utxos[0] = utxo;
        _mint(utxos);
    }

    function withdraw(
        uint256 amount,
        uint256[2] memory inputs,
        uint256 output,
        Commonlib.Proof calldata proof
    ) public {
        validateTransactionProposal(inputs, [output, 0], proof);
        _withdraw(amount, inputs, output, proof);
        processInputsAndOutputs(inputs, [output, 0]);
    }

    function mint(uint256[] memory utxos) public onlyOwner {
        _mint(utxos);
    }
}
