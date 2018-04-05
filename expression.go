package main
import "strconv"
import "errors"
import "strings"
import "log"

type Expression interface {
	Evaluate(map[string]interface{}) bool
	String() string
}

type Logical struct {
	Args []*Expression
	Name string
}

func (e Logical) Evaluate(ass map[string]interface{}) bool {
	if e.Name == "&&" {
		for _,a := range(e.Args) {
			if !(*a).Evaluate(ass) {
				return false
			}
		}
		return true
	}
	if e.Name == "||" {
		for _,a := range(e.Args) {
			if (*a).Evaluate(ass) {
				return true
			}
		}
		return false
	}
	panic("Logical op " + e.Name + " not supported")
}

func (e Logical) String() string {
	s := ""
	s += e.Name
	s += "("
	for i,el := range(e.Args) {
		s += (*el).String()
		if i != len((e).Args)-1 {
			s += ", "
		}
	}
	s += ")"
	return s
}

type Predicate struct {
	Args [] *Arg
	Name string
}

type Arg struct {
	name string
	val string
}

func (a Arg) String() string {
	if len(a.name) > 0 {
		return a.name
	} else {
		return a.val
	}
}

func (a Arg) Val(ass map[string]interface{}) interface{} {
	if len(a.name) > 0 {
		if v,ok := ass[a.name].(int64); ok { // cast ints to float
			return float64(v)		
		}
		if v,ok := ass[a.name].(int32); ok { // cast ints to float
			return float64(v)		
		}
		if v,ok := ass[a.name].(int); ok { // cast ints to float
			return float64(v)		
		}
		return ass[a.name]
	} else {
		if s, err := strconv.ParseFloat(a.val, 64); err == nil {
			return s
		}
		return a.val
	}
}

func (e Predicate) Evaluate(ass map[string]interface{}) bool {
	a0 := e.Args[0].Val(ass)
	if len(e.Args) < 2 {
		if b, ok := a0.(bool); ok {
			return b
		} else {
			panic("Nullary non-bool predicate")
		}
	} else {

		a1 := e.Args[1].Val(ass)
		
		switch va0 := a0.(type) {
			case float64:
				va1, ok := a1.(float64)
				if !ok {
					panic("Expected float for second argument")
				}
				switch e.Name {
					case ">":
						return va0 > va1
					case ">=":
						return va0 >= va1
					case "<":
						return va0 < va1
					case "<=":
						return va0 <= va1
					case "=":
						return va0 == va1
					default:
						panic("Operator " + e.Name + " not supported")
				}

			case string: // string
				va1, ok := a1.(string)
				if !ok {
					panic("Expected string for second argument")
				}
				switch e.Name {
					case "=":
						return va0 == va1
					case "contains":
						return strings.Contains(va0, va1)
					case "Contains":
						return strings.Contains(strings.ToLower(va0), strings.ToLower(va1))
					default:
						panic("Operator " + e.Name + " not supported")
				}

			default:
				//log.Print("%T", a1)
				panic("Type not supported")
		}
	}
}

func (e Predicate) String() string {
	if len(e.Args) == 2 {
		return e.Args[0].String() + " " + e.Name + " " + e.Args[1].String()
	} else {
		return e.Name
	}
}

// Parsing

func ParseAndEval(expr string, interpretation map[string]interface{}) (result bool, err error) {
	defer func() {
		if r:=recover(); r!=nil {
			log.Print(r)
			err = errors.New("Error occured during parsing/evaluation of " + expr + ": ")
		}
	}()
	_,e := ParseExpr(Tokenizer(expr))
	result = (*e).Evaluate(interpretation)
	err = nil
	return result, err
}

func ParseExpr(tokens []string) (int,*Expression)  {
	isLogicalOp := func(a string) bool { return a == "&&" || a == "||" }
	if len(tokens) < 2 || len(tokens) == 3 && !isLogicalOp(tokens[1]) {
		_,p := ParsePredicate(tokens)
		return 0,p
	}
	opName := ""
	expectOp := false
	e := Logical{}
	i := 0
	// always infix
	for i= 0; i<len(tokens); i++ {
		t := tokens[i]
		//log.Print(t)
		if t == ")" {
			i++
			break
		}
		if !expectOp {
			if t == "(" {
				j, exp := ParseExpr(tokens[i+1:])
				e.Args = append(e.Args, exp)
				i = j+i
			} else {
				//log.Print("Parse pred:" + tokens[i])
				j,subE := ParsePredicate(tokens[i:])
				i+=j
				e.Args = append(e.Args, subE)

			}
			expectOp = true // repeating op (a && b && c)
		} else if expectOp {
			if opName == "" {
				opName = t
				e.Name = opName
			} else if opName != t {
				log.Print(opName, t)
				panic("repeating ops not statisfied")
			}
			expectOp = false
		}
		
	}
	eCasted := Expression(e)
	return i,&(eCasted)
}

func ParseArg(arg string) *Arg  {
	p := Arg{}

	if arg[0] == '.' { // to be bound to val.
		varName := arg[1:]
		p.name = varName
	} else {
		p.val = arg
	}
	return &p
}
func ParsePredicate(tokens []string) (int,*Expression)  {
	p := Predicate{}
	isInfix := func(a string) bool { return strings.Contains(">>=<<=containsContains", a)}
	var e Expression	
	// infix predicate
	if len(tokens) > 2  && isInfix(tokens[1])  {
		p.Args = append(p.Args, ParseArg(tokens[0]))
		p.Args = append(p.Args, ParseArg(tokens[2]))
		p.Name = tokens[1]
		e = Expression(p)
		return 2,&e
	} else { // do nullary predicate
		p.Args = append(p.Args, ParseArg(tokens[0]))
		e = Expression(p)
		return 0,&e
	}
}

func Tokenizer(exp string) []string {
	var tokens []string
	paren := false
	cToken := ""
	for i:=0; i<len(exp); i++ {
		c := string(exp[i])
		if paren {
			if c == "\"" {
				paren = false
			} else {
				cToken += c
				continue
			}
		} else if !paren && c == "\"" {
			paren = true
		} else if c == "(" || c == ")" {
			if cToken != "" {
				tokens = append(tokens,cToken)
			}
			tokens = append(tokens,c)
			cToken = ""
		} else if c == " " {
			if cToken != "" {
				tokens = append(tokens,cToken)	
			}
			cToken = ""
		} else {
			cToken += c
		}
	}
	if cToken != "" {
		tokens = append(tokens,cToken)
	}
	return tokens
}
