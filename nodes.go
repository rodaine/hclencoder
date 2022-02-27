package hclencoder

import (
	"errors"
	"fmt"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	"reflect"
	"sort"
	"strings"
)

const (
	// HCLTagName is the struct field tag used by the HCL decoder. The
	// values from this tag are used in the same way as the decoder.
	HCLTagName = "hcl"

	// KeyTag indicates that the value of the field should be part of
	// the parent object block's key, not a property of that block
	KeyTag string = "key"

	// SquashTag is attached to fields of a struct and indicates
	// to the encoder to lift the fields of that value into the parent
	// block's scope transparently.
	SquashTag string = "squash"

	// Blocks is attached to a slice of objects and indicates that
	// the slice should be treated as multiple separate blocks rather than
	// a list.
	Blocks string = "blocks"

	// Expression indicates that this field should not be quoted.
	Expression string = "expr"

	// UnusedKeysTag is a flag that indicates any unused keys found by the
	// decoder are stored in this field of type []string. This has the same
	// behavior as the OmitTag and is not encoded.
	UnusedKeysTag string = "unusedKeys"

	// DecodedFieldsTag is a flag that indicates all fields decoded are
	// stored in this field of type []string. This has the same behavior as
	// the OmitTag and is not encoded.
	DecodedFieldsTag string = "decodedFields"

	// HCLETagName is the struct field tag used by this package. The
	// values from this tag are used in conjunction with HCLTag values.
	HCLETagName = "hcle"

	// OmitTag will omit this field from encoding. This is the similar
	// behavior to `json:"-"`.
	OmitTag string = "omit"

	// OmitEmptyTag will omit this field if it is a zero value. This
	// is similar behavior to `json:",omitempty"`
	OmitEmptyTag string = "omitempty"
)

type fieldMeta struct {
	anonymous     bool
	name          string
	key           bool
	squash        bool
	repeatBlock   bool
	expression    bool
	unusedKeys    bool
	decodedFields bool
	omit          bool
	omitEmpty     bool
}

func encode(in reflect.Value) (node *Node, err error) {
	return encodeField(in, fieldMeta{})
}

// encode converts a reflected valued into an HCL ast.Node in a depth-first manner.
func encodeField(in reflect.Value, meta fieldMeta) (node *Node, err error) {
	in, isNil := deref(in)
	if isNil {
		return nil, nil
	}

	switch in.Kind() {

	case reflect.Bool, reflect.Float64, reflect.String,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return encodePrimitive(in, meta)

	case reflect.Slice:
		return encodeList(in, meta)

	case reflect.Map:
		return encodePrimitive(in, meta)

	case reflect.Struct:
		return encodeStruct(in, meta)
	default:
		return nil, fmt.Errorf("cannot encode kind %s to HCL", in.Kind())
	}
}

// encodePrimitive converts a primitive value into a Node contains its tokens
func encodePrimitive(in reflect.Value, meta fieldMeta) (*Node, error) {
	// Keys must be literals, so we don't tokenize.
	if meta.key {
		k := cty.StringVal(in.String())
		return &Node{Value: &k}, nil
	}
	tkn, err := tokenize(in, meta)
	if err != nil {
		return nil, err
	}

	return &Node{Tokens: tkn}, nil
}

// encodeList converts a slice into either a block list or a primitive list depending on its element type
func encodeList(in reflect.Value, meta fieldMeta) (*Node, error) {
	childType := in.Type().Elem()

childLoop:
	for {
		switch childType.Kind() {
		case reflect.Ptr:
			childType = childType.Elem()
		default:
			break childLoop
		}
	}

	switch childType.Kind() {
	case reflect.Map, reflect.Struct, reflect.Interface:
		return encodeBlockList(in, meta)
	default:
		return encodePrimitiveList(in, meta)
	}
}

// encodePrimitiveList converts a slice of primitive values to an ast.ListType. An
// ast.ObjectKey is never returned.
func encodePrimitiveList(in reflect.Value, meta fieldMeta) (*Node, error) {
	return encodePrimitive(in, meta)
}

// encodeBlockList converts a slice of non-primitive types to an ast.ObjectList. An
// ast.ObjectKey is never returned.
func encodeBlockList(in reflect.Value, meta fieldMeta) (*Node, error) {
	var blocks []*hclwrite.Block

	if !meta.repeatBlock {
		return encodePrimitiveList(in, meta)
	}

	for i := 0; i < in.Len(); i++ {
		node, err := encodeStruct(in.Index(i), meta)
		if err != nil {
			return nil, err
		}
		if node == nil {
			continue
		}
		blocks = append(blocks, node.Block)
	}

	return &Node{BlockList: blocks}, nil
}

type Node struct {
	Block     *hclwrite.Block
	BlockList []*hclwrite.Block
	Value     *cty.Value
	Tokens    hclwrite.Tokens
}

func (n Node) isValue() bool {
	return n.Value != nil
}

func (n Node) isBlock() bool {
	return n.Block != nil
}

func (n Node) isBlockList() bool {
	return n.BlockList != nil
}

func (n Node) isTokens() bool {
	return n.Tokens != nil
}

// encodeStruct converts a struct type into a block
func encodeStruct(in reflect.Value, parentMeta fieldMeta) (*Node, error) {
	l := in.NumField()
	block := hclwrite.NewBlock(parentMeta.name, nil)

	for i := 0; i < l; i++ {
		field := in.Type().Field(i)
		meta := extractFieldMeta(field)

		// these tags are used for debugging the decoder
		// they should not be output
		if meta.unusedKeys || meta.decodedFields || meta.omit {
			continue
		}

		// if the OmitEmptyTag is provided, check if the value is its zero value.
		rawVal := in.Field(i)
		if meta.omitEmpty {
			zeroVal := reflect.Zero(rawVal.Type()).Interface()
			if reflect.DeepEqual(rawVal.Interface(), zeroVal) {
				continue
			}
		}

		val, err := encodeField(rawVal, meta)
		if err != nil {
			return nil, err
		}
		if val == nil {
			continue
		}

		// this field is a key and should be bubbled up to the parent node
		if meta.key {
			if val.isValue() && (*val.Value).Type() == cty.String {
				label := (*val.Value).AsString()
				block.SetLabels(append(block.Labels(), label))
				continue
			}
			return nil, errors.New("struct key fields must be string literals")
		}

		if meta.squash && !val.isBlock() {
			return nil, errors.New("squash fields must be structs")
		}

		if val.isBlock() {
			if meta.squash {
				SquashBlock(val.Block, block.Body())
				for _, label := range val.Block.Labels() {
					block.SetLabels(append(block.Labels(), label))
				}
			} else {
				block.Body().AppendBlock(val.Block)
			}
			continue
		} else if val.isBlockList() {
			for _, innerBlock := range val.BlockList {
				block.Body().AppendBlock(innerBlock)
			}
		} else if val.isValue() {
			block.Body().SetAttributeValue(meta.name, *val.Value)
		} else if val.isTokens() {
			block.Body().SetAttributeRaw(meta.name, val.Tokens)
		} else {
			return nil, errors.New("unknown value type")
		}

	}

	return &Node{Block: block}, nil
}

func SquashBlock(innerBlock *hclwrite.Block, block *hclwrite.Body) {
	tkns := innerBlock.Body().BuildTokens(nil)
	block.AppendUnstructuredTokens(tkns)

}

func convertTokens(tokens hclsyntax.Tokens) hclwrite.Tokens {
	var result []*hclwrite.Token
	for _, token := range tokens {
		result = append(result, &hclwrite.Token{
			Type:         token.Type,
			Bytes:        token.Bytes,
			SpacesBefore: 0,
		})
	}
	return result
}

// tokenize converts a primitive type into tokens. structs and maps are converted into objects and slices are converted
// into tuples.
func tokenize(in reflect.Value, meta fieldMeta) (tkns hclwrite.Tokens, err error) {

	tokenEqual := hclwrite.Token{
		Type:         hclsyntax.TokenEqual,
		Bytes:        []byte("="),
		SpacesBefore: 0,
	}
	tokenComma := hclwrite.Token{
		Type:         hclsyntax.TokenComma,
		Bytes:        []byte(","),
		SpacesBefore: 0,
	}
	tokenOCurlyBrace := hclwrite.Token{
		Type:         hclsyntax.TokenOBrace,
		Bytes:        []byte("{"),
		SpacesBefore: 0,
	}
	tokenCCurlyBrace := hclwrite.Token{
		Type:         hclsyntax.TokenCBrace,
		Bytes:        []byte("}"),
		SpacesBefore: 0,
	}

	switch in.Kind() {
	case reflect.Bool:
		return hclwrite.TokensForValue(cty.BoolVal(in.Bool())), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return hclwrite.TokensForValue(cty.NumberUIntVal(in.Uint())), nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return hclwrite.TokensForValue(cty.NumberIntVal(in.Int())), nil

	case reflect.Float64:
		return hclwrite.TokensForValue(cty.NumberFloatVal(in.Float())), nil

	case reflect.String:
		val := in.String()
		if !meta.expression {
			val = fmt.Sprintf(`"%s"`, EscapeString(val))
		}
		// Unfortunately hcl escapes template expressions (${...}) when using hclwrite.TokensForValue. So we escape
		// everything but template expressions and then parse the expression into tokens.
		tokens, diags := hclsyntax.LexExpression([]byte(val), meta.name, hcl.Pos{
			Line:   0,
			Column: 0,
			Byte:   0,
		})

		if diags != nil {
			return nil, fmt.Errorf("error when parsing string %s: %v", val, diags.Error())
		}
		return convertTokens(tokens), nil
	case reflect.Pointer, reflect.Interface:
		val, isNil := deref(in)
		if isNil {
			return nil, nil
		}
		return tokenize(val, meta)
	case reflect.Struct:
		var tokens []*hclwrite.Token
		tokens = append(tokens, &tokenOCurlyBrace)
		for i := 0; i < in.NumField(); i++ {
			field := in.Type().Field(i)
			meta := extractFieldMeta(field)

			rawVal := in.Field(i)
			if meta.omitEmpty {
				zeroVal := reflect.Zero(rawVal.Type()).Interface()
				if reflect.DeepEqual(rawVal.Interface(), zeroVal) {
					continue
				}
			}
			val, err := tokenize(rawVal, meta)
			if err != nil {
				return nil, err
			}
			for _, tkn := range hclwrite.TokensForValue(cty.StringVal(meta.name)) {
				tokens = append(tokens, tkn)
			}
			tokens = append(tokens, &tokenEqual)
			for _, tkn := range val {
				tokens = append(tokens, tkn)
			}
			if i < in.NumField()-1 {
				tokens = append(tokens, &tokenComma)
			}
		}
		tokens = append(tokens, &tokenCCurlyBrace)
		return tokens, nil
	case reflect.Slice:
		var tokens []*hclwrite.Token
		tokens = append(tokens, &hclwrite.Token{
			Type:         hclsyntax.TokenOBrace,
			Bytes:        []byte("["),
			SpacesBefore: 0,
		})
		for i := 0; i < in.Len(); i++ {
			value, err := tokenize(in.Index(i), meta)
			if err != nil {
				return nil, err
			}
			for _, tkn := range value {
				tokens = append(tokens, tkn)
			}
			if i < in.Len()-1 {
				tokens = append(tokens, &tokenComma)
			}
		}
		tokens = append(tokens, &hclwrite.Token{
			Type:         hclsyntax.TokenCBrace,
			Bytes:        []byte("]"),
			SpacesBefore: 0,
		})
		return tokens, nil
	case reflect.Map:
		if keyType := in.Type().Key().Kind(); keyType != reflect.String {
			return nil, fmt.Errorf("map keys must be strings, %s given", keyType)
		}
		var tokens []*hclwrite.Token
		tokens = append(tokens, &tokenOCurlyBrace)

		var keys []string
		for _, k := range in.MapKeys() {
			keys = append(keys, k.String())
		}
		sort.Strings(keys)
		for i, k := range keys {
			val, err := tokenize(in.MapIndex(reflect.ValueOf(k)), meta)
			if err != nil {
				return nil, err
			}
			for _, tkn := range hclwrite.TokensForValue(cty.StringVal(k)) {
				tokens = append(tokens, tkn)
			}
			tokens = append(tokens, &tokenEqual)
			for _, tkn := range val {
				tokens = append(tokens, tkn)
			}
			if i < len(keys)-1 {
				tokens = append(tokens, &tokenComma)
			}
		}
		tokens = append(tokens, &tokenCCurlyBrace)
		return tokens, nil
	}

	return nil, fmt.Errorf("cannot encode primitive kind %s to token", in.Kind())
}

// extractFieldMeta pulls information about struct fields and the optional HCL tags
func extractFieldMeta(f reflect.StructField) (meta fieldMeta) {
	if f.Anonymous {
		meta.anonymous = true
		meta.name = f.Type.Name()
	} else {
		meta.name = f.Name
	}

	tags := strings.Split(f.Tag.Get(HCLTagName), ",")
	if len(tags) > 0 {
		if tags[0] != "" {
			meta.name = tags[0]
		}

		for _, tag := range tags[1:] {
			switch tag {
			case KeyTag:
				meta.key = true
			case SquashTag:
				meta.squash = true
			case DecodedFieldsTag:
				meta.decodedFields = true
			case UnusedKeysTag:
				meta.unusedKeys = true
			case Blocks:
				meta.repeatBlock = true
			case Expression:
				meta.expression = true
			}
		}
	}

	tags = strings.Split(f.Tag.Get(HCLETagName), ",")
	for _, tag := range tags {
		switch tag {
		case OmitTag:
			meta.omit = true
		case OmitEmptyTag:
			meta.omitEmpty = true
		}
	}

	return
}

// deref safely dereferences interface and pointer values to their underlying value types.
// It also detects if that value is invalid or nil.
func deref(in reflect.Value) (val reflect.Value, isNil bool) {
	switch in.Kind() {
	case reflect.Invalid:
		return in, true
	case reflect.Interface, reflect.Ptr:
		if in.IsNil() {
			return in, true
		}
		// recurse for the elusive double pointer
		return deref(in.Elem())
	case reflect.Slice, reflect.Map:
		return in, in.IsNil()
	default:
		return in, false
	}
}
