package main

import (
	"github.com/latentart/gu/el"
)

func main() {
	el.Mount("#app", App)
	select {}
}
