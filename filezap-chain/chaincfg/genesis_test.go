// Copyright (c) 2014-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chaincfg

import (
	"bytes"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/require"
)

// TestGenesisBlock tests the genesis block of the main network for validity by
// checking the encoded bytes and hashes.
func TestGenesisBlock(t *testing.T) {
	// Encode the genesis block to raw bytes.
	var buf bytes.Buffer
	err := MainNetParams.GenesisBlock.Serialize(&buf)
	if err != nil {
		t.Fatalf("TestGenesisBlock: %v", err)
	}

	// Ensure the encoded block matches the expected bytes.
	if !bytes.Equal(buf.Bytes(), genesisBlockBytes) {
		t.Fatalf("TestGenesisBlock: Genesis block does not appear valid - "+
			"got %v, want %v", spew.Sdump(buf.Bytes()),
			spew.Sdump(genesisBlockBytes))
	}

	// Check hash of the block against expected hash.
	hash := MainNetParams.GenesisBlock.BlockHash()
	if !MainNetParams.GenesisHash.IsEqual(&hash) {
		t.Fatalf("TestGenesisBlock: Genesis block hash does not "+
			"appear valid - got %v, want %v", spew.Sdump(hash),
			spew.Sdump(MainNetParams.GenesisHash))
	}
}

// TestRegTestGenesisBlock tests the genesis block of the regression test
// network for validity by checking the encoded bytes and hashes.
func TestRegTestGenesisBlock(t *testing.T) {
	// Encode the genesis block to raw bytes.
	var buf bytes.Buffer
	err := RegressionNetParams.GenesisBlock.Serialize(&buf)
	if err != nil {
		t.Fatalf("TestRegTestGenesisBlock: %v", err)
	}

	// Ensure the encoded block matches the expected bytes.
	if !bytes.Equal(buf.Bytes(), regTestGenesisBlockBytes) {
		t.Fatalf("TestRegTestGenesisBlock: Genesis block does not "+
			"appear valid - got %v, want %v",
			spew.Sdump(buf.Bytes()),
			spew.Sdump(regTestGenesisBlockBytes))
	}

	// Check hash of the block against expected hash.
	hash := RegressionNetParams.GenesisBlock.BlockHash()
	if !RegressionNetParams.GenesisHash.IsEqual(&hash) {
		t.Fatalf("TestRegTestGenesisBlock: Genesis block hash does "+
			"not appear valid - got %v, want %v", spew.Sdump(hash),
			spew.Sdump(RegressionNetParams.GenesisHash))
	}
}

// TestTestNet3GenesisBlock tests the genesis block of the test network (version
// 3) for validity by checking the encoded bytes and hashes.
func TestTestNet3GenesisBlock(t *testing.T) {
	// Encode the genesis block to raw bytes.
	var buf bytes.Buffer
	err := TestNet3Params.GenesisBlock.Serialize(&buf)
	if err != nil {
		t.Fatalf("TestTestNet3GenesisBlock: %v", err)
	}

	// Ensure the encoded block matches the expected bytes.
	if !bytes.Equal(buf.Bytes(), testNet3GenesisBlockBytes) {
		t.Fatalf("TestTestNet3GenesisBlock: Genesis block does not "+
			"appear valid - got %v, want %v",
			spew.Sdump(buf.Bytes()),
			spew.Sdump(testNet3GenesisBlockBytes))
	}

	// Check hash of the block against expected hash.
	hash := TestNet3Params.GenesisBlock.BlockHash()
	if !TestNet3Params.GenesisHash.IsEqual(&hash) {
		t.Fatalf("TestTestNet3GenesisBlock: Genesis block hash does "+
			"not appear valid - got %v, want %v", spew.Sdump(hash),
			spew.Sdump(TestNet3Params.GenesisHash))
	}
}

// TestTestNet4GenesisBlock tests the genesis block of the test network (version
// 4) for validity by checking the encoded bytes and hashes.
func TestTestNet4GenesisBlock(t *testing.T) {
	// Encode the genesis block to raw bytes.
	var buf bytes.Buffer
	err := TestNet4Params.GenesisBlock.Serialize(&buf)
	require.NoError(t, err)

	// Ensure the encoded block matches the expected bytes.
	if !bytes.Equal(buf.Bytes(), testNet4GenesisBlockBytes) {
		t.Fatalf("TestTestNet4GenesisBlock: Genesis block does not "+
			"appear valid - got %v, want %v",
			spew.Sdump(buf.Bytes()),
			spew.Sdump(testNet4GenesisBlockBytes))
	}

	// Check hash of the block against expected hash.
	hash := TestNet4Params.GenesisBlock.BlockHash()
	if !TestNet4Params.GenesisHash.IsEqual(&hash) {
		t.Fatalf("TestTestNet4GenesisBlock: Genesis block hash does "+
			"not appear valid - got %v, want %v", spew.Sdump(hash),
			spew.Sdump(TestNet4Params.GenesisHash))
	}
	expectedHash := "00000000da84f2bafbbc53dee25a72ae507ff4914b867c565be3" +
		"50b0da8bf043"
	require.Equal(t, expectedHash, hash.String())
}

// TestSimNetGenesisBlock tests the genesis block of the simulation test network
// for validity by checking the encoded bytes and hashes.
func TestSimNetGenesisBlock(t *testing.T) {
	// Encode the genesis block to raw bytes.
	var buf bytes.Buffer
	err := SimNetParams.GenesisBlock.Serialize(&buf)
	if err != nil {
		t.Fatalf("TestSimNetGenesisBlock: %v", err)
	}

	// Ensure the encoded block matches the expected bytes.
	if !bytes.Equal(buf.Bytes(), simNetGenesisBlockBytes) {
		t.Fatalf("TestSimNetGenesisBlock: Genesis block does not "+
			"appear valid - got %v, want %v",
			spew.Sdump(buf.Bytes()),
			spew.Sdump(simNetGenesisBlockBytes))
	}

	// Check hash of the block against expected hash.
	hash := SimNetParams.GenesisBlock.BlockHash()
	if !SimNetParams.GenesisHash.IsEqual(&hash) {
		t.Fatalf("TestSimNetGenesisBlock: Genesis block hash does "+
			"not appear valid - got %v, want %v", spew.Sdump(hash),
			spew.Sdump(SimNetParams.GenesisHash))
	}
}

// TestSigNetGenesisBlock tests the genesis block of the signet test network for
// validity by checking the encoded bytes and hashes.
func TestSigNetGenesisBlock(t *testing.T) {
	// Encode the genesis block to raw bytes.
	var buf bytes.Buffer
	err := SigNetParams.GenesisBlock.Serialize(&buf)
	if err != nil {
		t.Fatalf("TestSigNetGenesisBlock: %v", err)
	}

	// Ensure the encoded block matches the expected bytes.
	if !bytes.Equal(buf.Bytes(), sigNetGenesisBlockBytes) {
		t.Fatalf("TestSigNetGenesisBlock: Genesis block does not "+
			"appear valid - got %v, want %v",
			spew.Sdump(buf.Bytes()),
			spew.Sdump(sigNetGenesisBlockBytes))
	}

	// Check hash of the block against expected hash.
	hash := SigNetParams.GenesisBlock.BlockHash()
	if !SigNetParams.GenesisHash.IsEqual(&hash) {
		t.Fatalf("TestSigNetGenesisBlock: Genesis block hash does "+
			"not appear valid - got %v, want %v", spew.Sdump(hash),
			spew.Sdump(SigNetParams.GenesisHash))
	}
}

// genesisBlockBytes are the wire encoded bytes for the genesis block of the
// main network as of protocol version 60002.
var genesisBlockBytes = []byte{
	0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x3b, 0xa3, 0xed, 0xfd, /* |....;...| */
	0x7a, 0x7b, 0x12, 0xb2, 0x7a, 0xc7, 0x2c, 0x3e, /* |z{..z.,>| */
	0x67, 0x76, 0x8f, 0x61, 0x7f, 0xc8, 0x1b, 0xc3, /* |gv.a....| */
	0x88, 0x8a, 0x51, 0x32, 0x3a, 0x9f, 0xb8, 0xaa, /* |..Q2:...| */
	0x4b, 0x1e, 0x5e, 0x4a, 0x29, 0xab, 0x5f, 0x49, /* |K.^J)._I| */
	0xff, 0xff, 0x00, 0x1d, 0x1d, 0xac, 0x2b, 0x7c, /* |......+|| */
	0x01, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, /* |........| */
	0xff, 0xff, 0x4d, 0x04, 0xff, 0xff, 0x00, 0x1d, /* |..M.....| */
	0x01, 0x04, 0x45, 0x54, 0x68, 0x65, 0x20, 0x54, /* |..EThe T| */
	0x69, 0x6d, 0x65, 0x73, 0x20, 0x30, 0x33, 0x2f, /* |imes 03/| */
	0x4a, 0x61, 0x6e, 0x2f, 0x32, 0x30, 0x30, 0x39, /* |Jan/2009| */
	0x20, 0x43, 0x68, 0x61, 0x6e, 0x63, 0x65, 0x6c, /* | Chancel| */
	0x6c, 0x6f, 0x72, 0x20, 0x6f, 0x6e, 0x20, 0x62, /* |lor on b| */
	0x72, 0x69, 0x6e, 0x6b, 0x20, 0x6f, 0x66, 0x20, /* |rink of | */
	0x73, 0x65, 0x63, 0x6f, 0x6e, 0x64, 0x20, 0x62, /* |second b| */
	0x61, 0x69, 0x6c, 0x6f, 0x75, 0x74, 0x20, 0x66, /* |ailout f| */
	0x6f, 0x72, 0x20, 0x62, 0x61, 0x6e, 0x6b, 0x73, /* |or banks| */
	0xff, 0xff, 0xff, 0xff, 0x01, 0x00, 0xf2, 0x05, /* |........| */
	0x2a, 0x01, 0x00, 0x00, 0x00, 0x43, 0x41, 0x04, /* |*....CA.| */
	0x67, 0x8a, 0xfd, 0xb0, 0xfe, 0x55, 0x48, 0x27, /* |g....UH'| */
	0x19, 0x67, 0xf1, 0xa6, 0x71, 0x30, 0xb7, 0x10, /* |.g..q0..| */
	0x5c, 0xd6, 0xa8, 0x28, 0xe0, 0x39, 0x09, 0xa6, /* |\..(.9..| */
	0x79, 0x62, 0xe0, 0xea, 0x1f, 0x61, 0xde, 0xb6, /* |yb...a..| */
	0x49, 0xf6, 0xbc, 0x3f, 0x4c, 0xef, 0x38, 0xc4, /* |I..?L.8.| */
	0xf3, 0x55, 0x04, 0xe5, 0x1e, 0xc1, 0x12, 0xde, /* |.U......| */
	0x5c, 0x38, 0x4d, 0xf7, 0xba, 0x0b, 0x8d, 0x57, /* |\8M....W| */
	0x8a, 0x4c, 0x70, 0x2b, 0x6b, 0xf1, 0x1d, 0x5f, /* |.Lp+k.._|*/
	0xac, 0x00, 0x00, 0x00, 0x00, /* |.....|    */
}

// regTestGenesisBlockBytes are the wire encoded bytes for the genesis block of
// the regression test network as of protocol version 60002.
var regTestGenesisBlockBytes = []byte{
	0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x3b, 0xa3, 0xed, 0xfd, /* |....;...| */
	0x7a, 0x7b, 0x12, 0xb2, 0x7a, 0xc7, 0x2c, 0x3e, /* |z{..z.,>| */
	0x67, 0x76, 0x8f, 0x61, 0x7f, 0xc8, 0x1b, 0xc3, /* |gv.a....| */
	0x88, 0x8a, 0x51, 0x32, 0x3a, 0x9f, 0xb8, 0xaa, /* |..Q2:...| */
	0x4b, 0x1e, 0x5e, 0x4a, 0xda, 0xe5, 0x49, 0x4d, /* |K.^J)._I| */
	0xff, 0xff, 0x7f, 0x20, 0x02, 0x00, 0x00, 0x00, /* |......+|| */
	0x01, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, /* |........| */
	0xff, 0xff, 0x4d, 0x04, 0xff, 0xff, 0x00, 0x1d, /* |..M.....| */
	0x01, 0x04, 0x45, 0x54, 0x68, 0x65, 0x20, 0x54, /* |..EThe T| */
	0x69, 0x6d, 0x65, 0x73, 0x20, 0x30, 0x33, 0x2f, /* |imes 03/| */
	0x4a, 0x61, 0x6e, 0x2f, 0x32, 0x30, 0x30, 0x39, /* |Jan/2009| */
	0x20, 0x43, 0x68, 0x61, 0x6e, 0x63, 0x65, 0x6c, /* | Chancel| */
	0x6c, 0x6f, 0x72, 0x20, 0x6f, 0x6e, 0x20, 0x62, /* |lor on b| */
	0x72, 0x69, 0x6e, 0x6b, 0x20, 0x6f, 0x66, 0x20, /* |rink of | */
	0x73, 0x65, 0x63, 0x6f, 0x6e, 0x64, 0x20, 0x62, /* |second b| */
	0x61, 0x69, 0x6c, 0x6f, 0x75, 0x74, 0x20, 0x66, /* |ailout f| */
	0x6f, 0x72, 0x20, 0x62, 0x61, 0x6e, 0x6b, 0x73, /* |or banks| */
	0xff, 0xff, 0xff, 0xff, 0x01, 0x00, 0xf2, 0x05, /* |........| */
	0x2a, 0x01, 0x00, 0x00, 0x00, 0x43, 0x41, 0x04, /* |*....CA.| */
	0x67, 0x8a, 0xfd, 0xb0, 0xfe, 0x55, 0x48, 0x27, /* |g....UH'| */
	0x19, 0x67, 0xf1, 0xa6, 0x71, 0x30, 0xb7, 0x10, /* |.g..q0..| */
	0x5c, 0xd6, 0xa8, 0x28, 0xe0, 0x39, 0x09, 0xa6, /* |\..(.9..| */
	0x79, 0x62, 0xe0, 0xea, 0x1f, 0x61, 0xde, 0xb6, /* |yb...a..| */
	0x49, 0xf6, 0xbc, 0x3f, 0x4c, 0xef, 0x38, 0xc4, /* |I..?L.8.| */
	0xf3, 0x55, 0x04, 0xe5, 0x1e, 0xc1, 0x12, 0xde, /* |.U......| */
	0x5c, 0x38, 0x4d, 0xf7, 0xba, 0x0b, 0x8d, 0x57, /* |\8M....W| */
	0x8a, 0x4c, 0x70, 0x2b, 0x6b, 0xf1, 0x1d, 0x5f, /* |.Lp+k.._|*/
	0xac, 0x00, 0x00, 0x00, 0x00, /* |.....|    */
}

// testNet3GenesisBlockBytes are the wire encoded bytes for the genesis block of
// the test network (version 3) as of protocol version 60002.
var testNet3GenesisBlockBytes = []byte{
	0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x3b, 0xa3, 0xed, 0xfd, /* |....;...| */
	0x7a, 0x7b, 0x12, 0xb2, 0x7a, 0xc7, 0x2c, 0x3e, /* |z{..z.,>| */
	0x67, 0x76, 0x8f, 0x61, 0x7f, 0xc8, 0x1b, 0xc3, /* |gv.a....| */
	0x88, 0x8a, 0x51, 0x32, 0x3a, 0x9f, 0xb8, 0xaa, /* |..Q2:...| */
	0x4b, 0x1e, 0x5e, 0x4a, 0xda, 0xe5, 0x49, 0x4d, /* |K.^J)._I| */
	0xff, 0xff, 0x00, 0x1d, 0x1a, 0xa4, 0xae, 0x18, /* |......+|| */
	0x01, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, /* |........| */
	0xff, 0xff, 0x4d, 0x04, 0xff, 0xff, 0x00, 0x1d, /* |..M.....| */
	0x01, 0x04, 0x45, 0x54, 0x68, 0x65, 0x20, 0x54, /* |..EThe T| */
	0x69, 0x6d, 0x65, 0x73, 0x20, 0x30, 0x33, 0x2f, /* |imes 03/| */
	0x4a, 0x61, 0x6e, 0x2f, 0x32, 0x30, 0x30, 0x39, /* |Jan/2009| */
	0x20, 0x43, 0x68, 0x61, 0x6e, 0x63, 0x65, 0x6c, /* | Chancel| */
	0x6c, 0x6f, 0x72, 0x20, 0x6f, 0x6e, 0x20, 0x62, /* |lor on b| */
	0x72, 0x69, 0x6e, 0x6b, 0x20, 0x6f, 0x66, 0x20, /* |rink of | */
	0x73, 0x65, 0x63, 0x6f, 0x6e, 0x64, 0x20, 0x62, /* |second b| */
	0x61, 0x69, 0x6c, 0x6f, 0x75, 0x74, 0x20, 0x66, /* |ailout f| */
	0x6f, 0x72, 0x20, 0x62, 0x61, 0x6e, 0x6b, 0x73, /* |or banks| */
	0xff, 0xff, 0xff, 0xff, 0x01, 0x00, 0xf2, 0x05, /* |........| */
	0x2a, 0x01, 0x00, 0x00, 0x00, 0x43, 0x41, 0x04, /* |*....CA.| */
	0x67, 0x8a, 0xfd, 0xb0, 0xfe, 0x55, 0x48, 0x27, /* |g....UH'| */
	0x19, 0x67, 0xf1, 0xa6, 0x71, 0x30, 0xb7, 0x10, /* |.g..q0..| */
	0x5c, 0xd6, 0xa8, 0x28, 0xe0, 0x39, 0x09, 0xa6, /* |\..(.9..| */
	0x79, 0x62, 0xe0, 0xea, 0x1f, 0x61, 0xde, 0xb6, /* |yb...a..| */
	0x49, 0xf6, 0xbc, 0x3f, 0x4c, 0xef, 0x38, 0xc4, /* |I..?L.8.| */
	0xf3, 0x55, 0x04, 0xe5, 0x1e, 0xc1, 0x12, 0xde, /* |.U......| */
	0x5c, 0x38, 0x4d, 0xf7, 0xba, 0x0b, 0x8d, 0x57, /* |\8M....W| */
	0x8a, 0x4c, 0x70, 0x2b, 0x6b, 0xf1, 0x1d, 0x5f, /* |.Lp+k.._|*/
	0xac, 0x00, 0x00, 0x00, 0x00, /* |.....|    */
}

// testNet4GenesisBlockBytes are the wire encoded bytes for the genesis block of
// the test network (version 4)
var testNet4GenesisBlockBytes = []byte{
	0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x4e, 0x7b, 0x2b, 0x91, /* |....N{+.| */
	0x28, 0xfe, 0x02, 0x91, 0xdb, 0x06, 0x93, 0xaf, /* |(.......| */
	0x2a, 0xe4, 0x18, 0xb7, 0x67, 0xe6, 0x57, 0xcd, /* |*...g.W.| */
	0x40, 0x7e, 0x80, 0xcb, 0x14, 0x34, 0x22, 0x1e, /* |@~...4".| */
	0xae, 0xa7, 0xa0, 0x7a, 0x04, 0x6f, 0x35, 0x66, /* |...z.o5f| */
	0xff, 0xff, 0x00, 0x1d, 0xbb, 0x0c, 0x78, 0x17, /* |......x.| */
	0x01, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, /* |........| */
	0xff, 0xff, 0x55, 0x04, 0xff, 0xff, 0x00, 0x1d, /* |..U.....| */
	0x01, 0x04, 0x4c, 0x4c, 0x30, 0x33, 0x2f, 0x4d, /* |..LL03/M| */
	0x61, 0x79, 0x2f, 0x32, 0x30, 0x32, 0x34, 0x20, /* |ay/2024 | */
	0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, /* |00000000| */
	0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, 0x30, /* |00000000| */
	0x30, 0x30, 0x30, 0x30, 0x31, 0x65, 0x62, 0x64, /* |00001ebd| */
	0x35, 0x38, 0x63, 0x32, 0x34, 0x34, 0x39, 0x37, /* |58c24497| */
	0x30, 0x62, 0x33, 0x61, 0x61, 0x39, 0x64, 0x37, /* |0b3aa9d7| */
	0x38, 0x33, 0x62, 0x62, 0x30, 0x30, 0x31, 0x30, /* |83bb0010| */
	0x31, 0x31, 0x66, 0x62, 0x65, 0x38, 0x65, 0x61, /* |11fbe8ea| */
	0x38, 0x65, 0x39, 0x38, 0x65, 0x30, 0x30, 0x65, /* |8e98e00e| */
	0xff, 0xff, 0xff, 0xff, 0x01, 0x00, 0xf2, 0x05, /* |........| */
	0x2a, 0x01, 0x00, 0x00, 0x00, 0x23, 0x21, 0x00, /* |*....#!.| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0xac, 0x00, 0x00, 0x00, 0x00, /* |.....   | */
}

// simNetGenesisBlockBytes are the wire encoded bytes for the genesis block of
// the simulation test network as of protocol version 70002.
var simNetGenesisBlockBytes = []byte{
	0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x3b, 0xa3, 0xed, 0xfd, /* |....;...| */
	0x7a, 0x7b, 0x12, 0xb2, 0x7a, 0xc7, 0x2c, 0x3e, /* |z{..z.,>| */
	0x67, 0x76, 0x8f, 0x61, 0x7f, 0xc8, 0x1b, 0xc3, /* |gv.a....| */
	0x88, 0x8a, 0x51, 0x32, 0x3a, 0x9f, 0xb8, 0xaa, /* |..Q2:...| */
	0x4b, 0x1e, 0x5e, 0x4a, 0x45, 0x06, 0x86, 0x53, /* |K.^J)._I| */
	0xff, 0xff, 0x7f, 0x20, 0x02, 0x00, 0x00, 0x00, /* |......+|| */
	0x01, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, /* |........| */
	0xff, 0xff, 0x4d, 0x04, 0xff, 0xff, 0x00, 0x1d, /* |..M.....| */
	0x01, 0x04, 0x45, 0x54, 0x68, 0x65, 0x20, 0x54, /* |..EThe T| */
	0x69, 0x6d, 0x65, 0x73, 0x20, 0x30, 0x33, 0x2f, /* |imes 03/| */
	0x4a, 0x61, 0x6e, 0x2f, 0x32, 0x30, 0x30, 0x39, /* |Jan/2009| */
	0x20, 0x43, 0x68, 0x61, 0x6e, 0x63, 0x65, 0x6c, /* | Chancel| */
	0x6c, 0x6f, 0x72, 0x20, 0x6f, 0x6e, 0x20, 0x62, /* |lor on b| */
	0x72, 0x69, 0x6e, 0x6b, 0x20, 0x6f, 0x66, 0x20, /* |rink of | */
	0x73, 0x65, 0x63, 0x6f, 0x6e, 0x64, 0x20, 0x62, /* |second b| */
	0x61, 0x69, 0x6c, 0x6f, 0x75, 0x74, 0x20, 0x66, /* |ailout f| */
	0x6f, 0x72, 0x20, 0x62, 0x61, 0x6e, 0x6b, 0x73, /* |or banks| */
	0xff, 0xff, 0xff, 0xff, 0x01, 0x00, 0xf2, 0x05, /* |........| */
	0x2a, 0x01, 0x00, 0x00, 0x00, 0x43, 0x41, 0x04, /* |*....CA.| */
	0x67, 0x8a, 0xfd, 0xb0, 0xfe, 0x55, 0x48, 0x27, /* |g....UH'| */
	0x19, 0x67, 0xf1, 0xa6, 0x71, 0x30, 0xb7, 0x10, /* |.g..q0..| */
	0x5c, 0xd6, 0xa8, 0x28, 0xe0, 0x39, 0x09, 0xa6, /* |\..(.9..| */
	0x79, 0x62, 0xe0, 0xea, 0x1f, 0x61, 0xde, 0xb6, /* |yb...a..| */
	0x49, 0xf6, 0xbc, 0x3f, 0x4c, 0xef, 0x38, 0xc4, /* |I..?L.8.| */
	0xf3, 0x55, 0x04, 0xe5, 0x1e, 0xc1, 0x12, 0xde, /* |.U......| */
	0x5c, 0x38, 0x4d, 0xf7, 0xba, 0x0b, 0x8d, 0x57, /* |\8M....W| */
	0x8a, 0x4c, 0x70, 0x2b, 0x6b, 0xf1, 0x1d, 0x5f, /* |.Lp+k.._|*/
	0xac, 0x00, 0x00, 0x00, 0x00, /* |.....|    */
}

// sigNetGenesisBlockBytes are the wire encoded bytes for the genesis block of
// the signet test network as of protocol version 70002.
var sigNetGenesisBlockBytes = []byte{
	0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |...@....| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x3b, 0xa3, 0xed, 0xfd, /* |........| */
	0x7a, 0x7b, 0x12, 0xb2, 0x7a, 0xc7, 0x2c, 0x3e, /* |....;...| */
	0x67, 0x76, 0x8f, 0x61, 0x7f, 0xc8, 0x1b, 0xc3, /* |z{..z.,>| */
	0x88, 0x8a, 0x51, 0x32, 0x3a, 0x9f, 0xb8, 0xaa, /* |gv.a....| */
	0x4b, 0x1e, 0x5e, 0x4a, 0x00, 0x8f, 0x4d, 0x5f, /* |..Q2:...| */
	0xae, 0x77, 0x03, 0x1e, 0x8a, 0xd2, 0x22, 0x03, /* |K.^J..M_| */
	0x01, 0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, /* |.w....".| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, /* |........| */
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff, /* |........| */
	0xff, 0xff, 0x4d, 0x04, 0xff, 0xff, 0x00, 0x1d, /* |........| */
	0x01, 0x04, 0x45, 0x54, 0x68, 0x65, 0x20, 0x54, /* |..M.....| */
	0x69, 0x6d, 0x65, 0x73, 0x20, 0x30, 0x33, 0x2f, /* |..EThe T| */
	0x4a, 0x61, 0x6e, 0x2f, 0x32, 0x30, 0x30, 0x39, /* |imes 03/| */
	0x20, 0x43, 0x68, 0x61, 0x6e, 0x63, 0x65, 0x6c, /* |Jan/2009| */
	0x6c, 0x6f, 0x72, 0x20, 0x6f, 0x6e, 0x20, 0x62, /* | Chancel| */
	0x72, 0x69, 0x6e, 0x6b, 0x20, 0x6f, 0x66, 0x20, /* |lor on b| */
	0x73, 0x65, 0x63, 0x6f, 0x6e, 0x64, 0x20, 0x62, /* |rink of| */
	0x61, 0x69, 0x6c, 0x6f, 0x75, 0x74, 0x20, 0x66, /* |second b| */
	0x6f, 0x72, 0x20, 0x62, 0x61, 0x6e, 0x6b, 0x73, /* |ailout f| */
	0xff, 0xff, 0xff, 0xff, 0x01, 0x00, 0xf2, 0x05, /* |or banks| */
	0x2a, 0x01, 0x00, 0x00, 0x00, 0x43, 0x41, 0x04, /* |........| */
	0x67, 0x8a, 0xfd, 0xb0, 0xfe, 0x55, 0x48, 0x27, /* |*....CA.| */
	0x19, 0x67, 0xf1, 0xa6, 0x71, 0x30, 0xb7, 0x10, /* |g....UH'| */
	0x5c, 0xd6, 0xa8, 0x28, 0xe0, 0x39, 0x09, 0xa6, /* |.g..q0..| */
	0x79, 0x62, 0xe0, 0xea, 0x1f, 0x61, 0xde, 0xb6, /* |\..(.9..| */
	0x49, 0xf6, 0xbc, 0x3f, 0x4c, 0xef, 0x38, 0xc4, /* |yb...a..| */
	0xf3, 0x55, 0x04, 0xe5, 0x1e, 0xc1, 0x12, 0xde, /* |I..?L.8.| */
	0x5c, 0x38, 0x4d, 0xf7, 0xba, 0x0b, 0x8d, 0x57, /* |.U......| */
	0x8a, 0x4c, 0x70, 0x2b, 0x6b, 0xf1, 0x1d, 0x5f, /* |\8M....W| */
	0xac, 0x00, 0x00, 0x00, 0x00, /* |.....| */
}
