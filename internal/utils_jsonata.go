package internal

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/blues/jsonata-go"
	"github.com/ghetzel/go-stockutil/typeutil"
)

func toJsonataExpr(def any) string {
	var exprstr string

	if typeutil.IsMap(def) {
		exprstr = typeutil.JSON(def)

		exprstr = regexp.MustCompile(`:\s*"([^"]+)"`).ReplaceAllString(exprstr, `: $1`)
	} else if typeutil.IsArray(def) {
		var segments []string

		for _, subdef := range typeutil.Slice(def) {
			segments = append(segments, toJsonataExpr(subdef.Value))
		}

		exprstr = strings.Join(segments, "")
	} else {
		exprstr = typeutil.String(def)
	}

	return exprstr
}

func makeJsonataExpressions(defs ...any) ([]*jsonata.Expr, error) {
	var jsonataExprs []*jsonata.Expr

	for _, def := range defs {
		var exprstr = toJsonataExpr(def)

		if exprstr == `` {
			continue
		}

		if expr, err := jsonata.Compile(exprstr); err == nil {
			jsonataExprs = append(jsonataExprs, expr)
		} else {
			return nil, fmt.Errorf("compile: %v", err)
		}
	}

	return jsonataExprs, nil
}

func applyJsonataExpressions(data any, vars map[string]any, exprs ...*jsonata.Expr) (any, error) {
	for _, expr := range exprs {
		if err := expr.RegisterVars(vars); err == nil {
			if out, err := expr.Eval(data); err == nil {
				data = out
			} else {
				return nil, fmt.Errorf("eval: %v", err)
			}
		} else {
			return nil, fmt.Errorf("vars: %v", err)
		}
	}

	return data, nil
}

func applyJsonata(data any, vars map[string]any, expr ...any) (any, error) {
	if ex, err := makeJsonataExpressions(expr...); err == nil {
		return applyJsonataExpressions(data, vars, ex...)
	} else {
		return nil, err
	}
}
