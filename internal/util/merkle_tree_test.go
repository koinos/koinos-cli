package util

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMerkleTree(t *testing.T) {
	values := []string{"the", "quick", "brown", "fox", "jumps", "over", "a", "lazy", "dog"}
	hasher := sha256.New()
	var hashes [][]byte

	for _, word := range values {
		hasher.Write([]byte(word))
		hashes = append(hashes, hasher.Sum(nil))
		hasher.Reset()
	}

	n01leaves := [][]byte{hashes[0], hashes[1]}
	n23leaves := [][]byte{hashes[2], hashes[3]}
	n0123leaves := append(n01leaves, n23leaves...)
	n45leaves := [][]byte{hashes[4], hashes[5]}
	n67leaves := [][]byte{hashes[6], hashes[7]}
	n4567leaves := append(n45leaves, n67leaves...)
	n01234567leaves := append(n0123leaves, n4567leaves...)
	n8leaves := [][]byte{hashes[8]}

	n01, _ := hex.DecodeString("0020397085ab4494829e691c49353a04d3201fda20c6a8a6866cf0f84bb8ce47")
	n23, _ := hex.DecodeString("78d4e37706320c82b2dd092eeb04b1f271523f86f910bf680ff9afcb2f8a33e1")
	n0123, _ := hex.DecodeString("e07aa684d91ffcbb89952f5e99b6181f7ee7bd88bd97be1345fc508f1062c050")
	n45, _ := hex.DecodeString("4185f41c5d7980ae7d14ce248f50e2854826c383671cf1ee3825ea957315c627")
	n67, _ := hex.DecodeString("b2a6704395c45ad8c99247103b580f7e7a37f06c3d38075ce4b02bc34c6a6754")
	n4567, _ := hex.DecodeString("2f24a249901ee8392ba0bb3b90c8efd6e2fee6530f45769199ef82d0b091d8ba")
	n01234567, _ := hex.DecodeString("913b7dce068efc8db6fab0173481f137ce91352b341855a1719aaff926169987")
	n8, _ := hex.DecodeString("cd6357efdd966de8c0cb2f876cc89ec74ce35f0968e11743987084bd42fb8944")
	merkleRoot, _ := hex.DecodeString("e24e552e0b6cf8835af179a14a766fb58c23e4ee1f7c6317d57ce39cc578cfac")

	assert.Equal(t, n01, CalculateMerkleRoot(n01leaves))
	assert.Equal(t, n23, CalculateMerkleRoot(n23leaves))
	assert.Equal(t, n0123, CalculateMerkleRoot(n0123leaves))
	assert.Equal(t, n45, CalculateMerkleRoot(n45leaves))
	assert.Equal(t, n67, CalculateMerkleRoot(n67leaves))
	assert.Equal(t, n4567, CalculateMerkleRoot(n4567leaves))
	assert.Equal(t, n01234567, CalculateMerkleRoot(n01234567leaves))
	assert.Equal(t, n8, CalculateMerkleRoot(n8leaves))
	assert.Equal(t, merkleRoot, CalculateMerkleRoot(hashes))
}
