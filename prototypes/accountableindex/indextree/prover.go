package indextree

type Prover struct {
	revealed map[*node]bool
	origins  map[*ProofNode]*node
	tree     Tree
}

func (t Tree) Prover() *Prover {
	return &Prover{tree: t, revealed: map[*node]bool{}, origins: map[*ProofNode]*node{}}
}

func (p *Prover) RevealRegionOf(word Word) {
	p.revealPathToward(wordPrefixOf(word), wordBits)
}

func (p *Prover) RevealLookupOf(key Key) {
	p.revealPathToward(key, keyBits)
}

func (p *Prover) revealPathToward(key Key, length int) {
	current := p.tree.root
	for current != nil {
		prefix, nodeLength := current.keyRange()
		if !key.sharesPrefixWith(prefix, min(nodeLength, length)) {
			return
		}
		if nodeLength >= length {
			if current.leaf != nil && current.leaf.Key == key {
				p.revealed[current] = true
			}

			return
		}
		p.revealed[current] = true
		current = current.childToward(key)
	}
}

func (p *Prover) Expand(stub *ProofNode) {
	if origin, found := p.origins[stub]; found {
		p.revealed[origin] = true
	}
}

func (p *Prover) Proof() Proof {
	clear(p.origins)

	return Proof{Root: p.proofNodeOf(p.tree.root)}
}

func (p *Prover) proofNodeOf(current *node) *ProofNode {
	if current == nil {
		return nil
	}
	prefix, length := current.keyRange()
	proofNode := &ProofNode{Prefix: prefix, Length: length, Hash: current.hash, Summary: current.summary}
	p.origins[proofNode] = current
	if !p.revealed[current] {
		return proofNode
	}
	proofNode.Revealed = true
	if current.leaf != nil {
		leaf := *current.leaf
		proofNode.Leaf = &leaf

		return proofNode
	}
	proofNode.Left = p.proofNodeOf(current.left)
	proofNode.Right = p.proofNodeOf(current.right)

	return proofNode
}
