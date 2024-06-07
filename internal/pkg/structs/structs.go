package structs

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/gob"
	"fmt"
	"strings"
	"time"

	"github.com/jenyaftw/trust/internal/pkg/utils"
)

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"

type NetworkNode struct {
	ID     int
	Status int
	IP     string
	Port   int
	Cert   *x509.Certificate
}

var Nodes []*NetworkNode

type TreeNode struct {
	Node  *NetworkNode
	Left  *TreeNode
	Right *TreeNode
}

func (n *TreeNode) String() string {
	return fmt.Sprintf("%d", n.Node.ID)
}

func (n *TreeNode) FillDeBruijn(max int, depth int) {
	bitCount := utils.GetBitCount(max)
	if depth > bitCount-1 {
		return
	}

	allMask, _, firstMask := utils.GetMasks(bitCount)
	first := (n.Node.ID >> 1) & allMask
	second := first | firstMask

	if first == n.Node.ID && second <= max {
		n.Left = &TreeNode{Node: Nodes[second]}
	} else if second == n.Node.ID && first <= max {
		n.Left = &TreeNode{Node: Nodes[first]}
	} else {
		if first <= max {
			n.Left = &TreeNode{Node: Nodes[first]}
		}
		if second <= max {
			n.Right = &TreeNode{Node: Nodes[second]}
			n.Right.FillDeBruijn(max, depth+1)
		}
	}
	n.Left.FillDeBruijn(max, depth+1)
}

func (n *TreeNode) CapturePrint(prefix string, isTail bool, initial bool, builder *strings.Builder) {
	if n.Right != nil {
		newPrefix := prefix
		if isTail {
			newPrefix += "│   "
		} else {
			newPrefix += "    "
		}
		n.Right.CapturePrint(newPrefix, false, false, builder)
	}

	builder.WriteString(prefix)
	if !initial {
		if isTail {
			builder.WriteString("└── ")
		} else {
			builder.WriteString("┌── ")
		}
	} else {
		builder.WriteString("    ")
	}
	color := Red
	if n.Node.Status == 1 {
		color = Yellow
	}
	if n.Node.Status == 2 {
		color = Green
	}
	builder.WriteString(fmt.Sprintf("%s%d%s\n", color, n.Node.ID, Reset))

	if n.Left != nil {
		newPrefix := prefix
		if isTail {
			newPrefix += "    "
		} else {
			newPrefix += "│   "
		}
		n.Left.CapturePrint(newPrefix, true, false, builder)
	}
}

func (n *TreeNode) PrintToString() string {
	var builder strings.Builder
	n.CapturePrint("", true, true, &builder)
	return builder.String()
}

func (n *TreeNode) FindNode(id int) *TreeNode {
	if n.Node.ID == id {
		return n
	}
	if n.Left != nil {
		if left := n.Left.FindNode(id); left != nil {
			return left
		}
	}
	if n.Right != nil {
		if right := n.Right.FindNode(id); right != nil {
			return right
		}
	}
	return nil
}

type Block struct {
	ID         int
	Timestamp  int64
	MerkleRoot []byte
	Data       []byte
	PrevHash   []byte
}

func (b *Block) CalculateHash() []byte {
	hash := sha256.New()
	hash.Write(b.Data)
	hash.Write(b.PrevHash)
	hash.Write([]byte(fmt.Sprint(b.ID)))
	hash.Write([]byte(fmt.Sprint(b.Timestamp)))
	return hash.Sum(nil)
}

func (b *Block) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(b)
	if err != nil {
		return nil, fmt.Errorf("error encoding message: %v", err)
	}

	return buf.Bytes(), nil
}

func DecodeBlock(data []byte) (*Block, error) {
	var block Block
	dec := gob.NewDecoder(bytes.NewReader(data))
	err := dec.Decode(&block)
	if err != nil {
		return nil, fmt.Errorf("error decoding block: %v", err)
	}

	return &block, nil
}

type Blockchain struct {
	Blocks []*Block
}

func NewBlockchain() *Blockchain {
	genesisBlock := &Block{
		ID:        0,
		Timestamp: 1111111111,
		Data:      []byte("Genesis block"),
		PrevHash:  nil,
	}

	genesisHash := genesisBlock.CalculateHash()
	genesisBlock.MerkleRoot = genesisHash

	blockchain := &Blockchain{
		Blocks: []*Block{genesisBlock},
	}

	return blockchain
}

func (bc *Blockchain) AddBlock(block *Block) *Block {
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	block.PrevHash = prevBlock.MerkleRoot
	block.ID = prevBlock.ID + 1
	block.Timestamp = time.Now().Unix()

	bc.Blocks = append(bc.Blocks, block)
	return block
}

func (bc *Blockchain) AddBlockFromBytes(data []byte) *Block {
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := &Block{
		ID:        prevBlock.ID + 1,
		Timestamp: time.Now().Unix(),
		Data:      data,
		PrevHash:  prevBlock.MerkleRoot,
	}

	bc.Blocks = append(bc.Blocks, newBlock)
	return newBlock
}

func (bc *Blockchain) Validate() bool {
	for i := 1; i < len(bc.Blocks); i++ {
		currentBlock := bc.Blocks[i]
		prevBlock := bc.Blocks[i-1]

		if string(currentBlock.PrevHash) != string(prevBlock.MerkleRoot) {
			return false
		}

		if string(currentBlock.MerkleRoot) != string(currentBlock.CalculateHash()) {
			return false
		}
	}
	return true
}

type MerkleNode struct {
	Left    *MerkleNode
	Right   *MerkleNode
	Value   []byte
	Content *Block
}

type MerkleTree struct {
	Root *MerkleNode
}

func BuildTreeFromBlockchain(bc *Blockchain) *MerkleTree {
	leaves := make([]*MerkleNode, 0)
	for _, block := range bc.Blocks {
		leaves = append(leaves, &MerkleNode{Value: block.CalculateHash(), Content: block})
	}

	if len(leaves)%2 != 0 {
		leaves = append(leaves, leaves[len(leaves)-1])
	}

	return &MerkleTree{
		Root: BuildTreeRecursively(leaves),
	}
}

func BuildTreeRecursively(nodes []*MerkleNode) *MerkleNode {
	if len(nodes)%2 == 1 {
		nodes = append(nodes, nodes[len(nodes)-1])
	}
	half := len(nodes) / 2

	if len(nodes) == 2 {
		hash := sha256.New()
		hash.Write(nodes[0].Value)
		hash.Write(nodes[1].Value)

		return &MerkleNode{
			Left:  nodes[0],
			Right: nodes[1],
			Value: hash.Sum(nil),
		}
	}

	left := BuildTreeRecursively(nodes[:half])
	right := BuildTreeRecursively(nodes[half:])

	hash := sha256.New()
	hash.Write(left.Value)
	hash.Write(right.Value)

	return &MerkleNode{
		Left:  left,
		Right: right,
		Value: hash.Sum(nil),
	}
}

func (mt *MerkleTree) AddBlock(block *Block) {
	newNode := &MerkleNode{
		Value:   block.CalculateHash(),
		Content: block,
	}

	if mt.Root == nil {
		mt.Root = newNode
		return
	}

	mt.Root = mt.AddNodeRecursively(newNode)
}

func (mt *MerkleTree) AddNodeRecursively(node *MerkleNode) *MerkleNode {
	if mt.Root == nil {
		return node
	}

	hash := sha256.New()
	hash.Write(mt.Root.Value)
	hash.Write(node.Value)

	newNode := &MerkleNode{
		Left:  mt.Root,
		Right: node,
		Value: hash.Sum(nil),
	}

	return mt.AddNodeRecursively(newNode)
}
