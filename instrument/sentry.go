package instrument

import (
	"go/ast"
	"go/token"
	"go/types"
)

type Sentry struct {
	TracerName  string
	ContextName string
	ErrorName   string

	hasInserts bool
	hasError   bool
}

func (s *Sentry) Imports() []*types.Package {
	if !s.hasInserts {
		return nil
	}
	return []*types.Package{
		types.NewPackage("github.com/MarketDataApp/marketdata-streamer/services/sentry", ""),
	}
}

func (s *Sentry) PrefixStatements(spanName string, hasError bool) []ast.Stmt {
	s.hasInserts = true
	if hasError {
		s.hasError = hasError
	}

	stmts := []ast.Stmt{
		&ast.AssignStmt{
			Tok: token.DEFINE,
			Lhs: []ast.Expr{&ast.Ident{Name: s.ContextName}, &ast.Ident{Name: "tracer"}},
			Rhs: []ast.Expr{s.expFuncSet(s.TracerName, spanName)},
		},
		&ast.DeferStmt{Call: &ast.CallExpr{
			Fun: &ast.FuncLit{
				Type: &ast.FuncType{},
				Body: &ast.BlockStmt{List: []ast.Stmt{
					&ast.ExprStmt{X: &ast.CallExpr{
						Fun: &ast.SelectorExpr{X: &ast.Ident{Name: "tracer"}, Sel: &ast.Ident{Name: "Finish"}},
						Args: []ast.Expr{
							&ast.Ident{Name: s.ContextName},
							&ast.Ident{Name: s.ErrorName},
						},
					}},
				}},
			},
		}},
	}
	return stmts
}

func (s *Sentry) expFuncSet(tracerName, spanName string) ast.Expr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   &ast.Ident{Name: "sentry"},
			Sel: &ast.Ident{Name: "Trace"},
		},
		Args: []ast.Expr{
			&ast.Ident{Name: s.ContextName},
			&ast.BasicLit{Kind: token.STRING, Value: `"` + tracerName + `"`},
			&ast.BasicLit{Kind: token.STRING, Value: `"` + spanName + `"`},
		},
	}
}