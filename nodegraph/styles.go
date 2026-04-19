package main

import "github.com/latentart/gu/el"

func GlobalStyles() el.Node {
	return el.Tag("style", el.Text(""))
}
