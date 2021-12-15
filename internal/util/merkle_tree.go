package util

import "crypto/sha256"

// CalculateMerkleRoot calculates the merkle root for given leafs
func CalculateMerkleRoot(nodes [][]byte) []byte {
	hasher := sha256.New()

	for len(nodes) > 1 {
		for i := 0; i < len(nodes); i += 2 {
			if i+1 < len(nodes) {
				hasher.Write(nodes[i])
				hasher.Write(nodes[i+1])
				nodes[i/2] = hasher.Sum(nil)
				hasher.Reset()
			} else {
				nodes[i/2] = nodes[i]
			}
		}

		nodes = nodes[:(len(nodes)+1)/2]
	}

	return nodes[0]
}
