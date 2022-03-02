package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

type CNF struct {
	head *clause
	tail *clause
}

type clause struct {
	Literals []int
	next     *clause
	prev     *clause
}

const ClauseEND = "0"

func (cnf *CNF) push(c *clause) {
	if cnf.head == nil && cnf.tail == nil {
		cnf.head = c
	} else {
		c.prev = cnf.tail
		c.prev.next = c
	}
	cnf.tail = c
}

func (cnf *CNF) delete(c *clause) {
	if c == cnf.head && c == cnf.tail {
		cnf.head = nil
		cnf.tail = nil
	} else if c == cnf.head {
		cnf.head = c.next
		c.next.prev = nil
	} else if c == cnf.tail {
		cnf.tail = c.prev
		c.prev.next = nil
	} else {
		c.prev.next, c.next.prev = c.next, c.prev
	}
}

func (c *clause) findIndex(literal int) (int, bool) {
	for index, l := range c.Literals {
		if l == literal {
			return index, true
		}
	}
	return -1, false
}

func (c *clause) remove(index int) {
	c.Literals = append(c.Literals[:index], c.Literals[index+1:]...)
}

func isSkipped(s string) bool {
	return len(s) == 0 || s[0] == '0' || s[0] == 'c' || s[0] == 'p' || s[0] == '%'
}

func (cnf *CNF) createClause(l []int) clause {
	return clause{Literals: l}
}

func (cnf *CNF) parseLiteral(s string) ([]int, error) {
	var literals = make([]int, 0, len(s)-1)

	for _, v := range strings.Fields(s) {
		if v == ClauseEND {
			break
		}

		num, err := strconv.Atoi(v)
		if err != nil {
			return nil, errors.New("wrong dimacs formats")
		}
		literals = append(literals, num)
	}

	if literals == nil {
		return nil, errors.New("wrong dimacs formats")
	}
	return literals, nil
}

func (cnf *CNF) Parse(f *os.File) error {
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		t := scanner.Text()
		if isSkipped(t) {
			continue
		}
		if literals, err := cnf.parseLiteral(t); err != nil {
			return err
		} else {
			clause := cnf.createClause(literals)
			cnf.push(&clause)
		}
	}
	return nil
}

func (cnf *CNF) deleteClause(target int) {
	for p := cnf.head; p != nil; p = p.next {
		if _, found := p.findIndex(target); found {
			cnf.delete(p)
		}
	}
}

func (cnf *CNF) deleteLiteral(target int) {
	for p := cnf.head; p != nil; p = p.next {
		if index, found := p.findIndex(target); found {
			p.remove(index)
		}
	}
}

/*
1リテラル規則（one literal rule, unit rule）
リテラル L 1つだけの節があれば、L を含む節を除去し、他の節の否定リテラル ¬L を消去する。
*/
func simplifyByUnitRule(cnf *CNF) {
	for p := cnf.head; p != nil; p = p.next {
		if len(p.Literals) == 1 {
			cnf.deleteClause(p.Literals[0])
			cnf.deleteLiteral(-p.Literals[0])
			p.next = cnf.head
		}
	}
}

/*
純リテラル規則（pure literal rule, affirmative-nagative rule）
節集合の中に否定と肯定の両方が現れないリテラル（純リテラル） L があれば、L を含む節を除去する。
*/
type purity struct {
	positive int
	negative int
}

func (cnf *CNF) getLiteralMap() map[int]purity {
	m := make(map[int]purity)

	for p := cnf.head; p != nil; p = p.next {
		for _, v := range p.Literals {
			newPurity := purity{}
			if old, ok := m[absInt(v)]; ok {
				newPurity = old
			}
			if v > 0 {
				newPurity.positive++
			} else {
				newPurity.negative++
			}
			m[absInt(v)] = newPurity
		}
	}
	return m
}

func (cnf *CNF) getPureClauseIndex(m map[int]purity) []int {

	res := []int{}
	for k, v := range m {
		if v.positive == 0 && v.negative > 0 {
			res = append(res, k)
		} else if v.positive > 0 && v.negative == 0 {
			res = append(res, -k)
		}
	}
	return res
}

func simplifyByPureRule(cnf *CNF) {
	literalMap := cnf.getLiteralMap()
	literals := cnf.getPureClauseIndex(literalMap)

	for _, v := range literals {
		cnf.deleteClause(v)
	}
}

func absInt(v int) int {
	return int(math.Abs(float64(v)))
}

func maxInteger(v1 int, v2 int) int {
	a := int(math.Max(float64(v1), float64(v2)))
	return a
}

// moms heuristicへの準備
func getAtomicFormula(cnf *CNF) int {
	literalsMap := cnf.getLiteralMap()

	maxNumber := 0
	maxInt := -1
	for k, v := range literalsMap {
		if v.positive > maxNumber || v.negative > maxNumber {
			maxNumber = maxInteger(v.positive, v.negative)
			maxInt = k
		}
	}
	return maxInt
}

func (cnf *CNF) deepCopy() *CNF {
	newcnf := &CNF{}
	for p := cnf.head; p != nil; p = p.next {
		newcnf.push(&clause{Literals: append([]int{}, p.Literals...)})
	}
	return newcnf
}

func (cnf *CNF) hasEmptyclause() bool {
	for p := cnf.head; p != nil; p = p.next {
		if len(p.Literals) == 0 {
			return true
		}
	}
	return false
}

func dpll(cnf *CNF) bool {
	simplifyByUnitRule(cnf)
	simplifyByPureRule(cnf)

	if cnf.head == nil {
		return true
	}

	if cnf.hasEmptyclause() {
		return false
	}

	variable := getAtomicFormula(cnf)
	cnfBranch1 := cnf.deepCopy()

	cnfBranch1.push(&clause{Literals: []int{variable}})
	if dpll(cnfBranch1) {
		return true
	}

	cnfBranch2 := cnf.deepCopy()
	cnfBranch2.push(&clause{Literals: []int{-variable}})
	return dpll(cnfBranch2)
}

func (cnf *CNF) IsSatisfied() bool {
	return dpll(cnf)
}

func main() {
	if len(os.Args) == 1 {
		cnf := &CNF{}
		if err := cnf.Parse(os.Stdin); err != nil {
			log.Fatal("Parse Error")
		}
		if cnf.IsSatisfied() {
			fmt.Println("sat")
		} else {
			fmt.Println("unsat")
		}
	} else {
		for i := 1; i < len(os.Args); i++ {
			f, err := os.Open(os.Args[i])
			if err != nil {
				log.Fatal("Parse Multi File Error")
			}
			defer f.Close()

			cnf := &CNF{}
			if err := cnf.Parse(f); err != nil {
				log.Fatal("Parse Error")
			}

			if cnf.IsSatisfied() {
				fmt.Println("sat")
			} else {
				fmt.Println("unsat")
			}
		}
	}
}