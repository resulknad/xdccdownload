package main

import "strings"
import "testing"
import "fmt"

func TestParseExpression(t *testing.T) {
	testString := "a && b && (c || \"d () a\" = a) && d"
	fmt.Println(strings.Join(Tokenizer(testString),"!"))

	if strings.Join(Tokenizer(testString),"!") != "a!&&!b!&&!(!c!||!d () a!=!a!)!&&!d" {
		t.FailNow()
	}
	
	testString = "b && ((d && a) || d) && d"
	fmt.Println(strings.Join(Tokenizer(testString),"!"))
	_,e := ParseExpr(Tokenizer(testString))
	fmt.Print((*e).String())
}

func TestEvalExpression(t *testing.T) {
	var a int32
	var b int64
	a = 2
	b = 2
	m := make(map[string]interface{})
	m["a"] = true
	m["b"] = a
	m["c"] = 3.4
	m["d"] = false
	m["dd"] = true
	m["somevar"] = true
	testString := ".a && .b > 1 && (.d || .dd)"

	fmt.Println(strings.Join(Tokenizer(testString),"!"))

	_,e := ParseExpr(Tokenizer(testString))
	if !(*e).Evaluate(m) {
		t.FailNow()
	}

	// ints will be cast to float64
	m["b"] = b
	if !(*e).Evaluate(m) {
		t.FailNow()
	}

	m["dd"] = false

	if (*e).Evaluate(m) {
		t.FailNow()
	}	
}

func TestParseAndEval(t *testing.T) {
	res, err := ParseAndEval(".a",map[string](interface{}){"a":true})

	if !res || err != nil {
		t.FailNow()
	}

	res, err = ParseAndEval(".a",map[string](interface{}){"a":false})
	if res || err != nil {
		t.FailNow()
	}

	res, err = ParseAndEval(".a && .a && (.a && .a && (.a && .a))",map[string](interface{}){"a":false})
	if res || err != nil {
		t.FailNow()
	}

	res, err = ParseAndEval(".a = \"some string\"",map[string](interface{}){"a":"some string"})
	if !res || err != nil {
		t.FailNow()
	}

	res, err = ParseAndEval(".a = \"some different string\"",map[string](interface{}){"a":"some string"})
	if res || err != nil {
		t.FailNow()
	}

	res, err = ParseAndEval(".a = \"some different string\" || .b = \"string\"",map[string](interface{}){"a":"some string", "b": "string"})
	if !res || err != nil {
		t.FailNow()
	}

	res, err = ParseAndEval(".a contains \"some string\" && aBcd Contains bc",map[string](interface{}){"a":"prefix: some string with postfix"})
	if !res || err != nil {
		t.FailNow()
	}

}
