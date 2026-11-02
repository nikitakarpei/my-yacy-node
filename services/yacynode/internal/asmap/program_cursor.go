package asmap

type instruction uint32

const (
	returnInstruction instruction = iota
	jumpInstruction
	matchInstruction
	defaultInstruction
)

const (
	minimumASN   = 1
	minimumJump  = 17
	minimumMatch = 2
)

var (
	instructionClassWidths = []uint{0, 0, 1}
	asnClassWidths         = []uint{15, 16, 17, 18, 19, 20, 21, 22, 23, 24}
	jumpClassWidths        = []uint{
		5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17,
		18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30,
	}
	matchClassWidths = []uint{1, 2, 3, 4, 5, 6, 7, 8}
)

type programCursor struct {
	program  []byte
	position uint32
}

func (c *programCursor) readInstruction() instruction {
	return instruction(c.readInteger(0, instructionClassWidths))
}

func (c *programCursor) readASN() ASN {
	return ASN(c.readInteger(minimumASN, asnClassWidths))
}

func (c *programCursor) readJump() uint32 {
	return c.readInteger(minimumJump, jumpClassWidths)
}

func (c *programCursor) readMatch() uint32 {
	return c.readInteger(minimumMatch, matchClassWidths)
}

func (c *programCursor) readInteger(minimum uint32, classWidths []uint) uint32 {
	integer := minimum
	lastClass := len(classWidths) - 1
	for _, width := range classWidths[:lastClass] {
		if !c.readBit() {
			return integer + c.readNumber(width)
		}
		integer += 1 << width
	}

	return integer + c.readNumber(classWidths[lastClass])
}

func (c *programCursor) readNumber(width uint) uint32 {
	var number uint32
	for range width {
		number <<= 1
		if c.readBit() {
			number |= 1
		}
	}

	return number
}

func (c *programCursor) readBit() bool {
	set := c.program[c.position/8]>>(c.position%8)&1 == 1
	c.position++

	return set
}

func (c *programCursor) skip(jump uint32) {
	c.position += jump
}
