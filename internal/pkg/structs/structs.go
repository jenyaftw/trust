package structs

import (
	"crypto/sha256"
	"crypto/x509"
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
	Hash       []byte
	PrevHash   []byte
}

func (b *Block) CalculateHash() []byte {
	record := fmt.Sprint(b.ID) + fmt.Sprint(b.Timestamp) + string(b.Data) + string(b.PrevHash)
	hash := sha256.New()
	// hash.Write(b.Data)
	// hash.Write(b.PrevHash)
	// hash.Write([]byte(string(b.ID)))
	// hash.Write([]byte(string(b.Timestamp)))
	hash.Write([]byte(record))
	return hash.Sum(nil)
}

type Blockchain struct {
	Blocks []*Block
}

func NewBlockchain() *Blockchain {
	genesisBlock := &Block{
		ID:        0,
		Timestamp: time.Now().Unix(),
		Data:      []byte("Genesis block"),
		PrevHash:  nil,
	}

	genesisHash := genesisBlock.CalculateHash()
	genesisBlock.Hash = genesisHash

	return &Blockchain{
		Blocks: []*Block{genesisBlock},
	}
}

func (bc *Blockchain) AddBlock(data []byte) {
	prevBlock := bc.Blocks[len(bc.Blocks)-1]
	newBlock := &Block{
		ID:        prevBlock.ID + 1,
		Timestamp: time.Now().Unix(),
		Data:      data,
		PrevHash:  prevBlock.Hash,
	}

	newBlock.Hash = newBlock.CalculateHash()
	bc.Blocks = append(bc.Blocks, newBlock)
}

func (bc *Blockchain) Validate() bool {
	for i := 1; i < len(bc.Blocks); i++ {
		currentBlock := bc.Blocks[i]
		prevBlock := bc.Blocks[i-1]

		if string(currentBlock.PrevHash) != string(prevBlock.Hash) {
			return false
		}

		if string(currentBlock.Hash) != string(currentBlock.CalculateHash()) {
			return false
		}
	}
	return true
}
