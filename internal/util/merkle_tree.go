package util

import (
	"crypto/sha256"

	"github.com/multiformats/go-multihash"
)

// CalculateMerkleRoot calculates the merkle root for given leafs
func CalculateMerkleRoot(nodes [][]byte) ([]byte, error) {
	hasher := sha256.New()

	for len(nodes) > 1 {
		for i := 0; i < len(nodes); i += 2 {
			if i+1 < len(nodes) {
				mHash, err := multihash.Decode(nodes[i])
				if err != nil {
					return nil, err
				}
				hasher.Write(mHash.Digest)

				mHash, err = multihash.Decode(nodes[i+1])
				if err != nil {
					return nil, err
				}
				hasher.Write(mHash.Digest)

				sum, err := multihash.Encode(hasher.Sum(nil), multihash.SHA2_256)
				if err != nil {
					return nil, err
				}

				nodes[i/2] = sum
				hasher.Reset()
			} else {
				nodes[i/2] = nodes[i]
			}
		}

		nodes = nodes[:(len(nodes)+1)/2]
	}

	return nodes[0], nil
}
