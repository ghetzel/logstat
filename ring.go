package main

type Ring struct {
    Size int
    Data []interface{}

    writeIdx   int
    writeCount int
}

func NewRing(size int) *Ring {
    ring := &Ring{
        Size: size,
    }

    ring.Clear()

    return ring
}

func (self *Ring) Clear() {
    self.writeCount = 0
    self.writeIdx = 0
    self.Data = make([]interface{}, self.Size)
}

func (self *Ring) Length() int {
    return len(self.Data)
}

func (self *Ring) WriteCount() int {
    return self.writeCount
}

func (self *Ring) Push(datum interface{}) {
    self.Data[self.writeIdx] = datum
    self.writeIdx = (self.writeIdx + 1) % self.Length()
    self.writeCount += 1
}

func (self *Ring) Seek(pos int) {
    self.writeIdx = (pos % self.Length())
}
