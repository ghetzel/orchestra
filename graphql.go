package orchestra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghetzel/go-stockutil/maputil"
	"github.com/ghetzel/go-stockutil/typeutil"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/formatter"
)

type GraphQLQuery struct {
	Name      string         `yaml:"name,omitempty"      json:"name,omitempty"`
	Query     map[string]any `yaml:"query"               json:"query"`
	Variables map[string]any `yaml:"variables,omitempty" json:"variables,omitempty"`
}

func (gql *GraphQLQuery) Render() (string, error) {
	var output bytes.Buffer

	var doc = new(ast.QueryDocument)
	var opdefs []map[string]any

	for qname, qdef := range gql.Query {
		var qmap = maputil.M(qdef).MapNative()
		var opdef = map[string]any{
			`Name`:      qname,
			`Operation`: `query`,
		}

		if vdef, err := gqlArglist(true, qmap[`@vars`]); err == nil {
			if vdef != nil {
				opdef[`VariableDefinitions`] = vdef
			}

			delete(qmap, `@vars`)
		} else {
			return ``, fmt.Errorf("bad variable: %v", err)
		}

		if sets, err := gql.parseSelectionSets(
			maputil.M(qdef).MapNative(),
		); err == nil {
			opdef[`SelectionSet`] = sets
		} else {
			return ``, err
		}

		opdefs = append(opdefs, opdef)
	}

	if encoded, err := json.Marshal(opdefs); err == nil {
		if err := json.Unmarshal(encoded, &doc.Operations); err != nil {
			return ``, fmt.Errorf("bad decode: %v", err)
		}
	} else {
		return ``, fmt.Errorf("bad def: %v", err)
	}

	formatter.NewFormatter(
		&output,
		formatter.WithIndent(`  `),
	).FormatQueryDocument(doc)

	return output.String(), nil
}

func (gql *GraphQLQuery) parseSelectionSets(cfgdef map[string]any) ([]map[string]any, error) {
	var selset []map[string]any

	for _, selname := range maputil.StringKeys(cfgdef) {
		var mapdef = make(map[string]any)
		var subdef = cfgdef[selname]
		var submap = maputil.M(subdef).MapNative()

		if !strings.HasPrefix(selname, `@`) {
			mapdef[`Alias`] = selname
			mapdef[`Name`] = selname

			if alias, ok := submap[`@alias`]; ok && alias != `` {
				mapdef[`Alias`] = alias
			}

			if vdef, err := gqlArglist(true, submap[`@vars`]); err == nil {
				if vdef != nil {
					mapdef[`VariableDefinitions`] = vdef
				}

				delete(submap, `@vars`)
			} else {
				return nil, fmt.Errorf("bad variable: %v", err)
			}

			if adef, err := gqlArglist(false, submap[`@args`]); err == nil {
				if adef != nil {
					mapdef[`Arguments`] = adef
				}

				delete(submap, `@args`)
			} else {
				return nil, fmt.Errorf("bad arg: %v", err)
			}

			if typeutil.IsMap(subdef) {
				if subset, err := gql.parseSelectionSets(submap); err == nil {
					mapdef[`SelectionSet`] = subset
				} else {
					return nil, fmt.Errorf("[%s] %v", selname, err)
				}
			} else if substr := typeutil.String(subdef); substr != `` {
				mapdef[`Name`] = substr
			}
		}

		if mapdef[`Name`] != `` {
			selset = append(selset, mapdef)
		}
	}

	return selset, nil
}

func gqlArglist(isvars bool, args any) ([]map[string]any, error) {
	if args == nil {
		return nil, nil
	} else if typeutil.IsMap(args) {
		args = []any{args}
	}

	if typeutil.IsArray(args) {
		var defs []map[string]any

		for _, vdef := range typeutil.Slice(args) {
			if vdef.IsMap() {
				for vname, mdef := range vdef.MapNative() {
					vname = strings.TrimSpace(vname)

					if isvars {
						vname = strings.TrimPrefix(vname, `$`)
						defs = append(defs, gqlVarValue(vname, typeutil.String(mdef)))
					} else {
						defs = append(defs, gqlArgValue(vname, mdef))
					}
				}
			}
		}

		return defs, nil
	} else {
		return nil, fmt.Errorf("definitions must be a list of objects or `name: type` strings")
	}
}

func gqlArgValue(vname string, vvalue any) map[string]any {
	switch rtype := gqlRawTypeDetect(vvalue); rtype {
	case ast.ObjectValue:
		var children []map[string]any

		for _, cname := range maputil.StringKeys(vvalue) {
			var cval = maputil.M(vvalue).Get(cname).Value
			children = append(children, gqlArgValue(cname, cval))
		}

		return map[string]any{
			`Name`: vname,
			`Value`: map[string]any{
				`Raw`:      ``,
				`Kind`:     int(rtype),
				`Children`: children,
			},
		}
	default:
		return map[string]any{
			`Name`: vname,
			`Value`: map[string]any{
				`Raw`:  strings.TrimPrefix(typeutil.String(vvalue), `$`),
				`Kind`: int(rtype),
			},
		}
	}
}

func gqlVarValue(vname string, vtype string) map[string]any {
	return map[string]any{
		`Variable`: vname,
		`Type`: map[string]any{
			`NamedType`: vtype,
		},
	}
}

func gqlRawTypeDetect(vvalue any) ast.ValueKind {
	var vstr = typeutil.String(vvalue)

	if vvalue == nil {
		return ast.NullValue
	} else if typeutil.IsMap(vvalue) {
		return ast.ObjectValue
	} else if typeutil.IsArray(vvalue) {
		return ast.ListValue
	} else if strings.HasPrefix(vstr, `$`) {
		return ast.Variable
	} else if typeutil.IsInteger(vvalue) {
		return ast.IntValue
	} else if typeutil.IsFloat(vvalue) {
		return ast.FloatValue
	} else if typeutil.IsKindOfBool(vvalue) {
		return ast.BooleanValue
	} else {
		return ast.StringValue
	}
}
