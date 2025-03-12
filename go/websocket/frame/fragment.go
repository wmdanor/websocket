package frame

func (f *Frame) IsControlFrame() bool {
	return f.Opcode.IsControl()
}

func (f *Frame) IsDataFrame() bool {
	return !f.Opcode.IsControl()
}

func (f *Frame) IsUnfragmentedDataFrame() bool {
	isFinal := f.FIN == FINFinalFrame
	isDataFrame := f.IsDataFrame()
	isContinuationFrame := f.Opcode == OpcodeContinuationFrame
	if isDataFrame && isFinal && !isContinuationFrame {
		return true
	} else {
		return false
	}
}

func (f *Frame) IsFirstFragmentDataFrame() bool {
	isFinal := f.FIN == FINFinalFrame
	isDataFrame := f.IsDataFrame()
	isContinuationFrame := f.Opcode == OpcodeContinuationFrame
	if isDataFrame && !isFinal && !isContinuationFrame {
		return true
	} else {
		return false
	}
}

func (f *Frame) IsMiddleFragmentDataFrame() bool {
	isFinal := f.FIN == FINFinalFrame
	isDataFrame := f.IsDataFrame()
	isContinuationFrame := f.Opcode == OpcodeContinuationFrame
	if isDataFrame && !isFinal && isContinuationFrame {
		return true
	} else {
		return false
	}
}

func (f *Frame) IsFinalFragmentDataFrame() bool {
	isFinal := f.FIN == FINFinalFrame
	isDataFrame := f.IsDataFrame()
	isContinuationFrame := f.Opcode == OpcodeContinuationFrame
	if isDataFrame && isFinal && isContinuationFrame {
		return true
	} else {
		return false
	}
}

func (f *Frame) IsFragmentedDataFrame() bool {
	if f.IsFirstFragmentDataFrame() {
		return true
	}
	if f.IsMiddleFragmentDataFrame() {
		return true
	}
	if f.IsFinalFragmentDataFrame() {
		return true
	}
	return false
}
