// Package exitcheck реализует анализатор для обнаружения использования
// panic, log.Fatal, os.Exit и zap.Logger.Fatal в коде.
package exitcheck

import (
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer проверяет использование panic, log.Fatal, os.Exit и zap.Logger.Fatal.
// Вызовы log.Fatal, os.Exit и zap.Logger.Fatal разрешены только в функции main пакета main.
var Analyzer = &analysis.Analyzer{
	Name:     "exitcheck",
	Doc:      "reports usage of panic, log.Fatal, os.Exit, and zap.Logger.Fatal outside of main() in main package",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	isMainPkg := pass.Pkg.Name() == "main"

	insp.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)

		inMainFunc := false
		if isMainPkg {
			inMainFunc = isInsideMainFunc(pass, call)
		}

		switch fn := call.Fun.(type) {
		case *ast.Ident:
			if fn.Name == "panic" {
				if obj := pass.TypesInfo.Uses[fn]; obj != nil {
					if _, ok := obj.(*types.Builtin); ok {
						pass.Reportf(call.Pos(), "usage of builtin panic is discouraged")
					}
				}
			}

		case *ast.SelectorExpr:
			sel := fn.Sel.Name

			if ident, ok := fn.X.(*ast.Ident); ok {
				obj := pass.TypesInfo.Uses[ident]
				if obj == nil {
					return
				}

				pkgName, ok := obj.(*types.PkgName)
				if !ok {
					return
				}

				pkgPath := pkgName.Imported().Path()

				if pkgPath == "log" && strings.HasPrefix(sel, "Fatal") {
					if isMainPkg && inMainFunc {
						return // Разрешено в main()
					}
					pass.Reportf(call.Pos(), "call to log.%s outside of main() in main package", sel)
				}

				if pkgPath == "os" && sel == "Exit" {
					if isMainPkg && inMainFunc {
						return // Разрешено в main()
					}
					pass.Reportf(call.Pos(), "call to os.Exit outside of main() in main package")
				}
			}

			if obj := pass.TypesInfo.Uses[fn.Sel]; obj != nil {
				if fnObj, ok := obj.(*types.Func); ok {
					sig := fnObj.Type().(*types.Signature)
					recv := sig.Recv()
					if recv == nil {
						return
					}

					recvType := recv.Type().String()
					if (strings.Contains(recvType, "go.uber.org/zap.Logger") ||
						strings.Contains(recvType, "go.uber.org/zap.SugaredLogger")) &&
						sel == "Fatal" {
						if isMainPkg && inMainFunc {
							return // Разрешено в main()
						}
						pass.Reportf(call.Pos(), "call to zap.Logger.Fatal outside of main() in main package")
					}
				}
			}
		}
	})

	return nil, nil
}

func isInsideMainFunc(pass *analysis.Pass, node ast.Node) bool {
	pos := node.Pos()

	for _, f := range pass.Files {
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			if fn.Name.Name == "main" && fn.Recv == nil {
				if fn.Body != nil && fn.Body.Pos() <= pos && pos <= fn.Body.End() {
					return true
				}
			}
		}
	}

	return false
}
