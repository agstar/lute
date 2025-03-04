// Lute - 一款结构化的 Markdown 引擎，支持 Go 和 JavaScript
// Copyright (c) 2019-present, b3log.org
//
// Lute is licensed under Mulan PSL v2.
// You can use this software according to the terms and conditions of the Mulan PSL v2.
// You may obtain a copy of Mulan PSL v2 at:
//         http://license.coscl.org.cn/MulanPSL2
// THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND, EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT, MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
// See the Mulan PSL v2 for more details.

package parse

import (
	"bytes"

	"github.com/agstar/lute/ast"
	"github.com/agstar/lute/lex"
)

func (context *Context) parseTable(paragraph *ast.Node) (retParagraph, retTable *ast.Node) {
	var tokens []byte
	length := len(paragraph.Tokens)
	lineCnt := 0
	for i := 0; i < length; i++ {
		if lex.ItemNewline == paragraph.Tokens[i] || 0 == i {
			if 0 == i {
				tokens = paragraph.Tokens[i:]
			} else {
				tokens = paragraph.Tokens[i+1:]
			}
			if table := context.parseTable0(tokens); nil != table {
				if 0 < lineCnt {
					retParagraph = &ast.Node{Type: ast.NodeParagraph, Tokens: paragraph.Tokens[0:i]}
				}
				retTable = table
				break
			}
		}
		lineCnt++
	}
	return
}

func (context *Context) parseTable0(tokens []byte) (ret *ast.Node) {
	lines := lex.Split(tokens, lex.ItemNewline)
	length := len(lines)
	if 2 > length {
		return
	}

	aligns := context.parseTableDelimRow(lex.TrimWhitespace(lines[1]))
	if nil == aligns {
		return
	}

	if 2 == length && 1 == len(aligns) && 0 == aligns[0] && !bytes.Contains(tokens, []byte("|")) {
		// 如果只有两行并且对齐方式是默认对齐且没有 | 时（foo\n---）就和 Setext 标题规则冲突了
		// 但在块级解析时显然已经尝试进行解析 Setext 标题，还能走到这里说明 Setetxt 标题解析失败，
		// 所以这里也不能当作表进行解析了，返回普通段落
		return
	}

	headRow := context.parseTableRow(lex.TrimWhitespace(lines[0]), aligns, true)
	if nil == headRow {
		return
	}

	ret = &ast.Node{Type: ast.NodeTable, TableAligns: aligns}
	ret.TableAligns = aligns
	ret.AppendChild(context.newTableHead(headRow))
	for i := 2; i < length; i++ {
		tableRow := context.parseTableRow(lex.TrimWhitespace(lines[i]), aligns, false)
		if nil == tableRow {
			return
		}
		ret.AppendChild(tableRow)
	}
	return
}

func (context *Context) newTableHead(headRow *ast.Node) *ast.Node {
	ret := &ast.Node{Type: ast.NodeTableHead}
	tr := &ast.Node{Type: ast.NodeTableRow}
	ret.AppendChild(tr)
	for c := headRow.FirstChild; nil != c; {
		next := c.Next
		tr.AppendChild(c)
		c = next
	}
	return ret
}

func (context *Context) parseTableRow(line []byte, aligns []int, isHead bool) (ret *ast.Node) {
	ret = &ast.Node{Type: ast.NodeTableRow, TableAligns: aligns}
	cols := lex.SplitWithoutBackslashEscape(line, lex.ItemPipe)
	if 1 > len(cols) {
		return nil
	}
	if lex.IsBlank(cols[0]) {
		cols = cols[1:]
	}
	if len(cols) > 0 && lex.IsBlank(cols[len(cols)-1]) {
		cols = cols[:len(cols)-1]
	}

	colsLen := len(cols)
	alignsLen := len(aligns)
	if isHead && colsLen > alignsLen { // 分隔符行定义了表的列数，如果表头列数还大于这个列数，则说明不满足表格式
		return nil
	}

	var i int
	var col []byte
	for ; i < colsLen && i < alignsLen; i++ {
		col = lex.TrimWhitespace(cols[i])
		cell := &ast.Node{Type: ast.NodeTableCell, TableCellAlign: aligns[i]}
		cell.Tokens = col
		ret.AppendChild(cell)
	}

	// 可能需要补全剩余的列
	for ; i < alignsLen; i++ {
		cell := &ast.Node{Type: ast.NodeTableCell, TableCellAlign: aligns[i]}
		ret.AppendChild(cell)
	}
	return
}

func (context *Context) parseTableDelimRow(line []byte) (aligns []int) {
	length := len(line)
	if 1 > length {
		return nil
	}

	var token byte
	var i int
	for ; i < length; i++ {
		token = line[i]
		if lex.ItemPipe != token && lex.ItemHyphen != token && lex.ItemColon != token && lex.ItemSpace != token {
			return nil
		}
	}

	cols := lex.SplitWithoutBackslashEscape(line, lex.ItemPipe)
	if lex.IsBlank(cols[0]) {
		cols = cols[1:]
	}
	if len(cols) > 0 && lex.IsBlank(cols[len(cols)-1]) {
		cols = cols[:len(cols)-1]
	}

	var alignments []int
	for _, col := range cols {
		col = lex.TrimWhitespace(col)
		if 1 > length || nil == col {
			return nil
		}

		align := context.tableDelimAlign(col)
		if -1 == align {
			return nil
		}
		alignments = append(alignments, align)
	}
	return alignments
}

func (context *Context) tableDelimAlign(col []byte) int {
	length := len(col)
	if 1 > length {
		return -1
	}

	var left, right bool
	first := col[0]
	left = lex.ItemColon == first
	last := col[length-1]
	right = lex.ItemColon == last

	i := 1
	var token byte
	for ; i < length-1; i++ {
		token = col[i]
		if lex.ItemHyphen != token {
			return -1
		}
	}

	if left && right {
		return 2
	}
	if left {
		return 1
	}
	if right {
		return 3
	}
	return 0
}
