package main

import "testing"

func TestHelloWorld(t *testing.T) {
    msg := "hello"
    if msg != "hello" {
    	t.Fatalf("Hello world test failed")
    }
}
