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

package integration_test

import (
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path"
	"testing"
	"time"

	"github.com/hyperledger-labs/zeto/go-sdk/internal/testutils"
	keyscore "github.com/hyperledger-labs/zeto/go-sdk/pkg/key-manager/core"
	"github.com/hyperledger-labs/zeto/go-sdk/pkg/key-manager/key"
	"github.com/hyperledger-labs/zeto/go-sdk/pkg/sparse-merkle-tree/node"
	"github.com/hyperledger-labs/zeto/go-sdk/pkg/sparse-merkle-tree/smt"
	"github.com/hyperledger-labs/zeto/go-sdk/pkg/sparse-merkle-tree/storage"
	"github.com/hyperledger-labs/zeto/go-sdk/pkg/utxo"
	"github.com/hyperledger/firefly-signer/pkg/keystorev3"
	"github.com/hyperledger/firefly-signer/pkg/secp256k1"
	"github.com/iden3/go-iden3-crypto/babyjub"
	"github.com/iden3/go-iden3-crypto/poseidon"
	"github.com/iden3/go-rapidsnark/prover"
	"github.com/iden3/go-rapidsnark/witness/v2"
	"github.com/iden3/go-rapidsnark/witness/wasmer"
	"github.com/stretchr/testify/assert"
)

const MAX_HEIGHT = 64

func loadCircuit(circuitName string) (witness.Calculator, []byte, error) {
	circuitRoot, exists := os.LookupEnv("CIRCUITS_ROOT")
	if !exists {
		return nil, []byte{}, fmt.Errorf("CIRCUITS_ROOT not set")
	}
	provingKeysRoot, exists := os.LookupEnv("PROVING_KEYS_ROOT")
	if !exists {
		return nil, []byte{}, fmt.Errorf("PROVING_KEYS_ROOT not set")
	}

	// load the wasm file for the circuit
	wasmBytes, err := os.ReadFile(path.Join(circuitRoot, fmt.Sprintf("%s_js", circuitName), fmt.Sprintf("%s.wasm", circuitName)))
	if err != nil {
		return nil, []byte{}, err
	}

	// create the prover
	zkeyBytes, err := os.ReadFile(path.Join(provingKeysRoot, fmt.Sprintf("%s.zkey", circuitName)))
	if err != nil {
		return nil, []byte{}, err
	}

	// create the calculator
	var ops []witness.Option
	ops = append(ops, witness.WithWasmEngine(wasmer.NewCircom2WitnessCalculator))
	calc, err := witness.NewCalculator(wasmBytes, ops...)
	if err != nil {
		return nil, []byte{}, err
	}

	return calc, zkeyBytes, err
}

func decryptKeyStorev3(t *testing.T) *secp256k1.KeyPair {
	// this would be read from a keystore file. The same file is used to persist
	// a private key for the secp256k1 curve
	const sampleWallet = `{
		"address": "5d093e9b41911be5f5c4cf91b108bac5d130fa83",
		"crypto": {
			"cipher": "aes-128-ctr",
			"ciphertext": "a28e5f6fd3189ef220f658392af0e967f17931530ac5b79376ed5be7d8adfb5a",
			"cipherparams": {
			"iv": "7babf856e25f812d9dbc133e3122a1fc"
			},
			"kdf": "scrypt",
			"kdfparams": {
			"dklen": 32,
			"n": 262144,
			"p": 1,
			"r": 8,
			"salt": "2844947e39e03785cad3ccda776279dbf5a86a5df9cb6d0ab5773bfcb7cbe3b7"
			},
			"mac": "69ed15cbb03a29ec194bdbd2c2d8084c62be620d5b3b0f668ed9aa1f45dbaf99"
		},
		"id": "307cc063-2344-426a-b992-3b72d5d5be0b",
		"version": 3
	}`

	w, err := keystorev3.ReadWalletFile([]byte(sampleWallet), []byte("correcthorsebatterystaple"))
	assert.NoError(t, err)
	keypair := w.KeyPair()
	return keypair
}

func testKeyFromKeyStorev3(t *testing.T) *keyscore.KeyEntry {
	keypair := decryptKeyStorev3(t)
	return key.NewKeyEntryFromPrivateKeyBytes([32]byte(keypair.PrivateKeyBytes()))
}

func TestZeto_1_SuccessfulProving(t *testing.T) {
	calc, provingKey, err := loadCircuit("anon")
	assert.NoError(t, err)
	assert.NotNil(t, calc)

	sender := testKeyFromKeyStorev3(t)
	receiver := testutils.NewKeypair()

	inputValues := []*big.Int{big.NewInt(30), big.NewInt(40)}
	outputValues := []*big.Int{big.NewInt(32), big.NewInt(38)}

	salt1 := utxo.NewSalt()
	input1, _ := poseidon.Hash([]*big.Int{inputValues[0], salt1, sender.PublicKey.X, sender.PublicKey.Y})
	salt2 := utxo.NewSalt()
	input2, _ := poseidon.Hash([]*big.Int{inputValues[1], salt2, sender.PublicKey.X, sender.PublicKey.Y})
	inputCommitments := []*big.Int{input1, input2}

	salt3 := utxo.NewSalt()
	output1, _ := poseidon.Hash([]*big.Int{outputValues[0], salt3, receiver.PublicKey.X, receiver.PublicKey.Y})
	salt4 := utxo.NewSalt()
	output2, _ := poseidon.Hash([]*big.Int{outputValues[1], salt4, sender.PublicKey.X, sender.PublicKey.Y})
	outputCommitments := []*big.Int{output1, output2}

	witnessInputs := map[string]interface{}{
		"inputCommitments":      inputCommitments,
		"inputValues":           inputValues,
		"inputSalts":            []*big.Int{salt1, salt2},
		"inputOwnerPrivateKey":  sender.PrivateKeyForZkp,
		"outputCommitments":     outputCommitments,
		"outputValues":          outputValues,
		"outputSalts":           []*big.Int{salt3, salt4},
		"outputOwnerPublicKeys": [][]*big.Int{{receiver.PublicKey.X, receiver.PublicKey.Y}, {sender.PublicKey.X, sender.PublicKey.Y}},
	}

	// calculate the witness object for checking correctness
	witness, err := calc.CalculateWitness(witnessInputs, true)
	assert.NoError(t, err)
	assert.NotNil(t, witness)

	assert.Equal(t, 0, witness[0].Cmp(big.NewInt(1)))
	assert.Equal(t, 0, witness[1].Cmp(inputCommitments[0]))
	assert.Equal(t, 0, witness[2].Cmp(inputCommitments[1]))
	assert.Equal(t, 0, witness[3].Cmp(outputCommitments[0]))
	assert.Equal(t, 0, witness[4].Cmp(outputCommitments[1]))

	// generate the witness binary to feed into the prover
	startTime := time.Now()
	witnessBin, err := calc.CalculateWTNSBin(witnessInputs, true)
	assert.NoError(t, err)
	assert.NotNil(t, witnessBin)

	proof, err := prover.Groth16Prover(provingKey, witnessBin)
	elapsedTime := time.Since(startTime)
	fmt.Printf("Proving time: %s\n", elapsedTime)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(proof.Proof.A))
	assert.Equal(t, 3, len(proof.Proof.B))
	assert.Equal(t, 3, len(proof.Proof.C))
	assert.Equal(t, 4, len(proof.PubSignals))
}

func TestZeto_2_SuccessfulProving(t *testing.T) {
	calc, provingKey, err := loadCircuit("anon_enc")
	assert.NoError(t, err)
	assert.NotNil(t, calc)

	sender := testKeyFromKeyStorev3(t)
	receiver := testutils.NewKeypair()

	inputValues := []*big.Int{big.NewInt(30), big.NewInt(40)}
	outputValues := []*big.Int{big.NewInt(32), big.NewInt(38)}

	salt1 := utxo.NewSalt()
	input1, _ := poseidon.Hash([]*big.Int{inputValues[0], salt1, sender.PublicKey.X, sender.PublicKey.Y})
	salt2 := utxo.NewSalt()
	input2, _ := poseidon.Hash([]*big.Int{inputValues[1], salt2, sender.PublicKey.X, sender.PublicKey.Y})
	inputCommitments := []*big.Int{input1, input2}

	salt3 := utxo.NewSalt()
	output1, _ := poseidon.Hash([]*big.Int{outputValues[0], salt3, receiver.PublicKey.X, receiver.PublicKey.Y})
	salt4 := utxo.NewSalt()
	output2, _ := poseidon.Hash([]*big.Int{outputValues[1], salt4, sender.PublicKey.X, sender.PublicKey.Y})
	outputCommitments := []*big.Int{output1, output2}

	encryptionNonce := utxo.NewSalt()

	witnessInputs := map[string]interface{}{
		"inputCommitments":      inputCommitments,
		"inputValues":           inputValues,
		"inputSalts":            []*big.Int{salt1, salt2},
		"inputOwnerPrivateKey":  sender.PrivateKeyForZkp,
		"outputCommitments":     outputCommitments,
		"outputValues":          outputValues,
		"outputSalts":           []*big.Int{salt3, salt4},
		"outputOwnerPublicKeys": [][]*big.Int{{receiver.PublicKey.X, receiver.PublicKey.Y}, {sender.PublicKey.X, sender.PublicKey.Y}},
		"encryptionNonce":       encryptionNonce,
	}

	startTime := time.Now()
	witnessBin, err := calc.CalculateWTNSBin(witnessInputs, true)
	assert.NoError(t, err)
	assert.NotNil(t, witnessBin)

	proof, err := prover.Groth16Prover(provingKey, witnessBin)
	elapsedTime := time.Since(startTime)
	fmt.Printf("Proving time: %s\n", elapsedTime)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(proof.Proof.A))
	assert.Equal(t, 3, len(proof.Proof.B))
	assert.Equal(t, 3, len(proof.Proof.C))
	assert.Equal(t, 7, len(proof.PubSignals))
}

func TestZeto_3_SuccessfulProving(t *testing.T) {
	calc, provingKey, err := loadCircuit("anon_nullifier")
	assert.NoError(t, err)
	assert.NotNil(t, calc)

	sender := testutils.NewKeypair()
	receiver := testutils.NewKeypair()

	inputValues := []*big.Int{big.NewInt(30), big.NewInt(40)}
	outputValues := []*big.Int{big.NewInt(32), big.NewInt(38)}

	salt1 := utxo.NewSalt()
	input1, _ := poseidon.Hash([]*big.Int{inputValues[0], salt1, sender.PublicKey.X, sender.PublicKey.Y})
	salt2 := utxo.NewSalt()
	input2, _ := poseidon.Hash([]*big.Int{inputValues[1], salt2, sender.PublicKey.X, sender.PublicKey.Y})
	inputCommitments := []*big.Int{input1, input2}

	nullifier1, _ := poseidon.Hash([]*big.Int{inputValues[0], salt1, sender.PrivateKeyBigInt})
	nullifier2, _ := poseidon.Hash([]*big.Int{inputValues[1], salt2, sender.PrivateKeyBigInt})
	nullifiers := []*big.Int{nullifier1, nullifier2}

	mt, err := smt.NewMerkleTree(storage.NewMemoryStorage(), MAX_HEIGHT)
	assert.NoError(t, err)
	utxo1 := node.NewFungible(inputValues[0], sender.PublicKey, salt1)
	n1, err := node.NewLeafNode(utxo1)
	assert.NoError(t, err)
	err = mt.AddLeaf(n1)
	assert.NoError(t, err)
	utxo2 := node.NewFungible(inputValues[1], sender.PublicKey, salt2)
	n2, err := node.NewLeafNode(utxo2)
	assert.NoError(t, err)
	err = mt.AddLeaf(n2)
	assert.NoError(t, err)
	proof1, _, err := mt.GenerateProof(input1, nil)
	assert.NoError(t, err)
	circomProof1, err := proof1.ToCircomVerifierProof(input1, input1, mt.Root(), MAX_HEIGHT)
	assert.NoError(t, err)
	proof2, _, err := mt.GenerateProof(input2, nil)
	assert.NoError(t, err)
	circomProof2, err := proof2.ToCircomVerifierProof(input2, input2, mt.Root(), MAX_HEIGHT)
	assert.NoError(t, err)

	salt3 := utxo.NewSalt()
	output1, _ := poseidon.Hash([]*big.Int{outputValues[0], salt3, receiver.PublicKey.X, receiver.PublicKey.Y})
	salt4 := utxo.NewSalt()
	output2, _ := poseidon.Hash([]*big.Int{outputValues[1], salt4, sender.PublicKey.X, sender.PublicKey.Y})
	outputCommitments := []*big.Int{output1, output2}

	proof1Siblings := make([]*big.Int, len(circomProof1.Siblings)-1)
	for i, s := range circomProof1.Siblings[0 : len(circomProof1.Siblings)-1] {
		proof1Siblings[i] = s.BigInt()
	}
	proof2Siblings := make([]*big.Int, len(circomProof2.Siblings)-1)
	for i, s := range circomProof2.Siblings[0 : len(circomProof2.Siblings)-1] {
		proof2Siblings[i] = s.BigInt()
	}
	witnessInputs := map[string]interface{}{
		"nullifiers":            nullifiers,
		"inputCommitments":      inputCommitments,
		"inputValues":           inputValues,
		"inputSalts":            []*big.Int{salt1, salt2},
		"inputOwnerPrivateKey":  sender.PrivateKeyBigInt,
		"root":                  mt.Root().BigInt(),
		"merkleProof":           [][]*big.Int{proof1Siblings, proof2Siblings},
		"enabled":               []*big.Int{big.NewInt(1), big.NewInt(1)},
		"outputCommitments":     outputCommitments,
		"outputValues":          outputValues,
		"outputSalts":           []*big.Int{salt3, salt4},
		"outputOwnerPublicKeys": [][]*big.Int{{receiver.PublicKey.X, receiver.PublicKey.Y}, {sender.PublicKey.X, sender.PublicKey.Y}},
	}

	startTime := time.Now()
	witnessBin, err := calc.CalculateWTNSBin(witnessInputs, true)
	assert.NoError(t, err)
	assert.NotNil(t, witnessBin)

	proof, err := prover.Groth16Prover(provingKey, witnessBin)
	elapsedTime := time.Since(startTime)
	fmt.Printf("Proving time: %s\n", elapsedTime)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(proof.Proof.A))
	assert.Equal(t, 3, len(proof.Proof.B))
	assert.Equal(t, 3, len(proof.Proof.C))
	assert.Equal(t, 7, len(proof.PubSignals))
}

func TestZeto_4_SuccessfulProving(t *testing.T) {
	calc, provingKey, err := loadCircuit("anon_enc_nullifier")
	assert.NoError(t, err)
	assert.NotNil(t, calc)

	sender := testutils.NewKeypair()
	receiver := testutils.NewKeypair()

	inputValues := []*big.Int{big.NewInt(30), big.NewInt(40)}
	outputValues := []*big.Int{big.NewInt(32), big.NewInt(38)}

	salt1 := utxo.NewSalt()
	input1, _ := poseidon.Hash([]*big.Int{inputValues[0], salt1, sender.PublicKey.X, sender.PublicKey.Y})
	salt2 := utxo.NewSalt()
	input2, _ := poseidon.Hash([]*big.Int{inputValues[1], salt2, sender.PublicKey.X, sender.PublicKey.Y})
	inputCommitments := []*big.Int{input1, input2}

	nullifier1, _ := poseidon.Hash([]*big.Int{inputValues[0], salt1, sender.PrivateKeyBigInt})
	nullifier2, _ := poseidon.Hash([]*big.Int{inputValues[1], salt2, sender.PrivateKeyBigInt})
	nullifiers := []*big.Int{nullifier1, nullifier2}

	mt, err := smt.NewMerkleTree(storage.NewMemoryStorage(), MAX_HEIGHT)
	assert.NoError(t, err)
	utxo1 := node.NewFungible(inputValues[0], sender.PublicKey, salt1)
	n1, err := node.NewLeafNode(utxo1)
	assert.NoError(t, err)
	err = mt.AddLeaf(n1)
	assert.NoError(t, err)
	utxo2 := node.NewFungible(inputValues[1], sender.PublicKey, salt2)
	n2, err := node.NewLeafNode(utxo2)
	assert.NoError(t, err)
	err = mt.AddLeaf(n2)
	assert.NoError(t, err)
	proof1, _, err := mt.GenerateProof(input1, nil)
	assert.NoError(t, err)
	circomProof1, err := proof1.ToCircomVerifierProof(input1, input1, mt.Root(), MAX_HEIGHT)
	assert.NoError(t, err)
	proof2, _, err := mt.GenerateProof(input2, nil)
	assert.NoError(t, err)
	circomProof2, err := proof2.ToCircomVerifierProof(input2, input2, mt.Root(), MAX_HEIGHT)
	assert.NoError(t, err)

	salt3 := utxo.NewSalt()
	output1, _ := poseidon.Hash([]*big.Int{outputValues[0], salt3, receiver.PublicKey.X, receiver.PublicKey.Y})
	salt4 := utxo.NewSalt()
	output2, _ := poseidon.Hash([]*big.Int{outputValues[1], salt4, sender.PublicKey.X, sender.PublicKey.Y})
	outputCommitments := []*big.Int{output1, output2}

	encryptionNonce := utxo.NewSalt()

	proof1Siblings := make([]*big.Int, len(circomProof1.Siblings)-1)
	for i, s := range circomProof1.Siblings[0 : len(circomProof1.Siblings)-1] {
		proof1Siblings[i] = s.BigInt()
	}
	proof2Siblings := make([]*big.Int, len(circomProof2.Siblings)-1)
	for i, s := range circomProof2.Siblings[0 : len(circomProof2.Siblings)-1] {
		proof2Siblings[i] = s.BigInt()
	}
	witnessInputs := map[string]interface{}{
		"nullifiers":            nullifiers,
		"inputCommitments":      inputCommitments,
		"inputValues":           inputValues,
		"inputSalts":            []*big.Int{salt1, salt2},
		"inputOwnerPrivateKey":  sender.PrivateKeyBigInt,
		"root":                  mt.Root().BigInt(),
		"merkleProof":           [][]*big.Int{proof1Siblings, proof2Siblings},
		"enabled":               []*big.Int{big.NewInt(1), big.NewInt(1)},
		"outputCommitments":     outputCommitments,
		"outputValues":          outputValues,
		"outputSalts":           []*big.Int{salt3, salt4},
		"outputOwnerPublicKeys": [][]*big.Int{{receiver.PublicKey.X, receiver.PublicKey.Y}, {sender.PublicKey.X, sender.PublicKey.Y}},
		"encryptionNonce":       encryptionNonce,
	}

	startTime := time.Now()
	witnessBin, err := calc.CalculateWTNSBin(witnessInputs, true)
	assert.NoError(t, err)
	assert.NotNil(t, witnessBin)

	proof, err := prover.Groth16Prover(provingKey, witnessBin)
	elapsedTime := time.Since(startTime)
	fmt.Printf("Proving time: %s\n", elapsedTime)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(proof.Proof.A))
	assert.Equal(t, 3, len(proof.Proof.B))
	assert.Equal(t, 3, len(proof.Proof.C))
	assert.Equal(t, 10, len(proof.PubSignals))
}

func TestZeto_5_SuccessfulProving(t *testing.T) {
	calc, provingKey, err := loadCircuit("nf_anon")
	assert.NoError(t, err)
	assert.NotNil(t, calc)

	sender := testutils.NewKeypair()
	receiver := testutils.NewKeypair()

	tokenId := big.NewInt(1001)
	tokenUri, err := utxo.HashTokenUri("https://example.com/token/1001")
	assert.NoError(t, err)

	salt1 := utxo.NewSalt()
	input1, err := poseidon.Hash([]*big.Int{tokenId, tokenUri, salt1, sender.PublicKey.X, sender.PublicKey.Y})
	assert.NoError(t, err)

	salt3 := utxo.NewSalt()
	output1, err := poseidon.Hash([]*big.Int{tokenId, tokenUri, salt3, receiver.PublicKey.X, receiver.PublicKey.Y})
	assert.NoError(t, err)

	witnessInputs := map[string]interface{}{
		"tokenIds":              []*big.Int{tokenId},
		"tokenUris":             []*big.Int{tokenUri},
		"inputCommitments":      []*big.Int{input1},
		"inputSalts":            []*big.Int{salt1},
		"inputOwnerPrivateKey":  sender.PrivateKeyBigInt,
		"outputCommitments":     []*big.Int{output1},
		"outputSalts":           []*big.Int{salt3},
		"outputOwnerPublicKeys": [][]*big.Int{{receiver.PublicKey.X, receiver.PublicKey.Y}},
	}

	// calculate the witness object for checking correctness
	witness, err := calc.CalculateWitness(witnessInputs, true)
	assert.NoError(t, err)
	assert.NotNil(t, witness)

	assert.Equal(t, 0, witness[0].Cmp(big.NewInt(1)))
	assert.Equal(t, 0, witness[1].Cmp(input1))
	assert.Equal(t, 0, witness[2].Cmp(output1))
	assert.Equal(t, 0, witness[3].Cmp(tokenId))
	assert.Equal(t, 0, witness[4].Cmp(tokenUri))

	// generate the witness binary to feed into the prover
	startTime := time.Now()
	witnessBin, err := calc.CalculateWTNSBin(witnessInputs, true)
	assert.NoError(t, err)
	assert.NotNil(t, witnessBin)

	proof, err := prover.Groth16Prover(provingKey, witnessBin)
	elapsedTime := time.Since(startTime)
	fmt.Printf("Proving time: %s\n", elapsedTime)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(proof.Proof.A))
	assert.Equal(t, 3, len(proof.Proof.B))
	assert.Equal(t, 3, len(proof.Proof.C))
	assert.Equal(t, 2, len(proof.PubSignals))
}

func TestZeto_6_SuccessfulProving(t *testing.T) {
	calc, provingKey, err := loadCircuit("nf_anon_nullifier")
	assert.NoError(t, err)
	assert.NotNil(t, calc)

	sender := testutils.NewKeypair()
	receiver := testutils.NewKeypair()

	tokenId := big.NewInt(1001)
	uriString := "https://example.com/token/1001"
	tokenUri, err := utxo.HashTokenUri(uriString)
	assert.NoError(t, err)

	salt1 := utxo.NewSalt()
	input1, err := poseidon.Hash([]*big.Int{tokenId, tokenUri, salt1, sender.PublicKey.X, sender.PublicKey.Y})
	assert.NoError(t, err)

	nullifier1, _ := poseidon.Hash([]*big.Int{tokenId, tokenUri, salt1, sender.PrivateKeyBigInt})

	mt, err := smt.NewMerkleTree(storage.NewMemoryStorage(), MAX_HEIGHT)
	assert.NoError(t, err)
	utxo1 := node.NewNonFungible(tokenId, uriString, sender.PublicKey, salt1)
	n1, err := node.NewLeafNode(utxo1)
	assert.NoError(t, err)
	err = mt.AddLeaf(n1)
	assert.NoError(t, err)
	proof1, _, err := mt.GenerateProof(input1, nil)
	assert.NoError(t, err)
	circomProof1, err := proof1.ToCircomVerifierProof(input1, input1, mt.Root(), MAX_HEIGHT)
	assert.NoError(t, err)
	proof1Siblings := make([]*big.Int, len(circomProof1.Siblings)-1)
	for i, s := range circomProof1.Siblings[0 : len(circomProof1.Siblings)-1] {
		proof1Siblings[i] = s.BigInt()
	}

	salt3 := utxo.NewSalt()
	output1, err := poseidon.Hash([]*big.Int{tokenId, tokenUri, salt3, receiver.PublicKey.X, receiver.PublicKey.Y})
	assert.NoError(t, err)

	witnessInputs := map[string]interface{}{
		"tokenId":              tokenId,
		"tokenUri":             tokenUri,
		"nullifier":            nullifier1,
		"inputCommitment":      input1,
		"inputSalt":            salt1,
		"inputOwnerPrivateKey": sender.PrivateKeyBigInt,
		"root":                 mt.Root().BigInt(),
		"merkleProof":          proof1Siblings,
		"outputCommitment":     output1,
		"outputSalt":           salt3,
		"outputOwnerPublicKey": []*big.Int{receiver.PublicKey.X, receiver.PublicKey.Y},
	}

	// calculate the witness object for checking correctness
	witness, err := calc.CalculateWitness(witnessInputs, true)
	assert.NoError(t, err)
	assert.NotNil(t, witness)

	assert.Equal(t, 0, witness[0].Cmp(big.NewInt(1)))
	assert.Equal(t, 0, witness[1].Cmp(nullifier1))

	// generate the witness binary to feed into the prover
	startTime := time.Now()
	witnessBin, err := calc.CalculateWTNSBin(witnessInputs, true)
	assert.NoError(t, err)
	assert.NotNil(t, witnessBin)

	proof, err := prover.Groth16Prover(provingKey, witnessBin)
	elapsedTime := time.Since(startTime)
	fmt.Printf("Proving time: %s\n", elapsedTime)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(proof.Proof.A))
	assert.Equal(t, 3, len(proof.Proof.B))
	assert.Equal(t, 3, len(proof.Proof.C))
	assert.Equal(t, 3, len(proof.PubSignals))
}

func TestConcurrentLeafnodesInsertion(t *testing.T) {
	x, _ := new(big.Int).SetString("9198063289874244593808956064764348354864043212453245695133881114917754098693", 10)
	y, _ := new(big.Int).SetString("3600411115173311692823743444460566395943576560299970643507632418781961416843", 10)
	alice := &babyjub.PublicKey{
		X: x,
		Y: y,
	}

	values := []int{10, 20, 30, 40}
	salts := []string{
		"43c49e8ba68a9b8a6bb5c230a734d8271a83d2f63722e7651272ebeef5446e",
		"19b965f7629e4f0c4bd0b8f9c87f17580f18a32a31b4641550071ee4916bbbfc",
		"9b0b93df975547e430eabff085a77831b8fcb6b5396e6bb815fda8d14125370",
		"194ec10ec96a507c7c9b60df133d13679b874b0bd6ab89920135508f55b3f064",
	}

	// run the test 10 times
	for i := 0; i < 100; i++ {
		// shuffle the utxos for this run
		for i := range values {
			j := rand.Intn(i + 1)
			values[i], values[j] = values[j], values[i]
			salts[i], salts[j] = salts[j], salts[i]
		}

		testConcurrentInsertion(t, alice, values, salts)
	}
}

func testConcurrentInsertion(t *testing.T, alice *babyjub.PublicKey, values []int, salts []string) {
	mt, err := smt.NewMerkleTree(storage.NewMemoryStorage(), MAX_HEIGHT)
	assert.NoError(t, err)
	done := make(chan bool, len(values))

	for i, v := range values {
		go func(i, v int) {
			salt, _ := new(big.Int).SetString(salts[i], 16)
			utxo := node.NewFungible(big.NewInt(int64(v)), alice, salt)
			n, err := node.NewLeafNode(utxo)
			assert.NoError(t, err)
			err = mt.AddLeaf(n)
			assert.NoError(t, err)
			done <- true
		}(i, v)
	}

	for i := 0; i < len(values); i++ {
		<-done
	}

	assert.Equal(t, "abacf46f5217552ee28fe50b8fd7ca6aa46daeb9acf9f60928654c3b1a472f23", mt.Root().Hex())
}

func TestKeyManager(t *testing.T) {
	keypair := decryptKeyStorev3(t)

	keyEntry := key.NewKeyEntryFromPrivateKeyBytes([32]byte(keypair.PrivateKeyBytes()))
	assert.NotNil(t, keyEntry)

	assert.NotNil(t, keyEntry.PrivateKey)
	assert.NotNil(t, keyEntry.PublicKey)
	assert.NotNil(t, keyEntry.PrivateKeyForZkp)
}
