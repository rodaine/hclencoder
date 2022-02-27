package hclencoder

import (
	"errors"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"reflect"
)

// Encode converts any supported type into the corresponding HCL format
func Encode(in interface{}) ([]byte, error) {
	node, err := encode(reflect.ValueOf(in))
	if err != nil {
		return nil, err
	}

	f := hclwrite.NewEmptyFile()
	if node.isBlock() {
		addRootBlock(node.Block, f)
	} else if node.isBlockList() {
		for _, block := range node.BlockList {
			f.Body().AppendBlock(block)
		}
	} else {
		return nil, errors.New("invalid root type - needs to be a block or block list")
	}

	return hclwrite.Format(f.Bytes()), nil
}

func addRootBlock(block *hclwrite.Block, f *hclwrite.File) {
	// root blocks without types are squashed by default
	if block.Type() == "" {
		SquashBlock(block, f.Body())
	} else {
		f.Body().AppendBlock(block)
	}
}
