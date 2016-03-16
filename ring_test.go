package main

import (
    "testing"
)

func TestCreateRing(t *testing.T) {
    ring := NewRing(10)

    if ring.Length() != 10 {
        t.Errorf("Failed: expected len=10, got len=%d", ring.Length())
    }
}

func TestRingPushSeek(t *testing.T) {
    ring := NewRing(4)

    ring.Push(1)
    ring.Push(2)
    ring.Push(3)
    ring.Push(4)

    shouldBe1 := []interface{}{ 1, 2, 3, 4 }
    shouldBe2 := []interface{}{ 5, 6, 7, 8 }
    shouldBe3 := []interface{}{ 5, 6, 7, 9 }

    for i, val := range ring.Data {
        if shouldBe1[i] != val {
            t.Errorf("Slice incorrect: should be %+v, got %+v", shouldBe1, ring.Data)
        }
    }

    ring.Push(5)
    ring.Push(6)
    ring.Push(7)
    ring.Push(8)

    for i, val := range ring.Data {
        if shouldBe2[i] != val {
            t.Errorf("Slice incorrect: should be %+v, got %+v", shouldBe2, ring.Data)
        }
    }

    ring.Seek(3)
    ring.Push(9)

    for i, val := range ring.Data {
        if shouldBe3[i] != val {
            t.Errorf("Slice incorrect: should be %+v, got %+v", shouldBe3, ring.Data)
        }
    }
}


func TestRingClear(t *testing.T) {
    ring := NewRing(4)

    ring.Push(1)
    ring.Push(2)
    ring.Push(3)
    ring.Push(4)
    ring.Push(5)

    shouldBe1 := []interface{}{ 5, 2, 3, 4 }
    shouldBe2 := []interface{}{ 6, 7, 8, nil }

    for i, val := range ring.Data {
        if shouldBe1[i] != val {
            t.Errorf("Slice incorrect: should be %+v, got %+v", shouldBe1, ring.Data)
        }
    }

    ring.Clear()

    ring.Push(6)
    ring.Push(7)
    ring.Push(8)

    for i, val := range ring.Data {
        if shouldBe2[i] != val {
            t.Errorf("Slice incorrect: should be %+v, got %+v", shouldBe2, ring.Data)
        }
    }
}