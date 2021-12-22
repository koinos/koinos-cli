package util

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
)

func TestMerkleTree(t *testing.T) {
	values := []string{"the", "quick", "brown", "fox", "jumps", "over", "a", "lazy", "dog"}
	hasher := sha256.New()
	var hashes [][]byte

	for _, word := range values {
		hasher.Write([]byte(word))
		mh, _ := multihash.Encode(hasher.Sum(nil), multihash.SHA2_256)
		hashes = append(hashes, mh)
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

	n01 := makeTestRoot("0020397085ab4494829e691c49353a04d3201fda20c6a8a6866cf0f84bb8ce47")
	n23 := makeTestRoot("78d4e37706320c82b2dd092eeb04b1f271523f86f910bf680ff9afcb2f8a33e1")
	n0123 := makeTestRoot("e07aa684d91ffcbb89952f5e99b6181f7ee7bd88bd97be1345fc508f1062c050")
	n45 := makeTestRoot("4185f41c5d7980ae7d14ce248f50e2854826c383671cf1ee3825ea957315c627")
	n67 := makeTestRoot("b2a6704395c45ad8c99247103b580f7e7a37f06c3d38075ce4b02bc34c6a6754")
	n4567 := makeTestRoot("2f24a249901ee8392ba0bb3b90c8efd6e2fee6530f45769199ef82d0b091d8ba")
	n01234567 := makeTestRoot("913b7dce068efc8db6fab0173481f137ce91352b341855a1719aaff926169987")
	n8 := makeTestRoot("cd6357efdd966de8c0cb2f876cc89ec74ce35f0968e11743987084bd42fb8944")
	merkleRoot := makeTestRoot("e24e552e0b6cf8835af179a14a766fb58c23e4ee1f7c6317d57ce39cc578cfac")

	checkLeaves(t, n01, n01leaves)
	checkLeaves(t, n23, n23leaves)
	checkLeaves(t, n0123, n0123leaves)
	checkLeaves(t, n45, n45leaves)
	checkLeaves(t, n67, n67leaves)
	checkLeaves(t, n4567, n4567leaves)
	checkLeaves(t, n01234567, n01234567leaves)
	checkLeaves(t, n8, n8leaves)
	checkLeaves(t, merkleRoot, hashes)
}

func makeTestRoot(s string) []byte {
	b, _ := hex.DecodeString(s)
	n, _ := multihash.Encode(b, multihash.SHA2_256)
	return n
}

func checkLeaves(t *testing.T, expected []byte, leaves [][]byte) {
	root, err := CalculateMerkleRoot(leaves)
	assert.NoError(t, err)
	assert.Equal(t, expected, root)
}
