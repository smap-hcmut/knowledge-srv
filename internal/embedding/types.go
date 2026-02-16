package embedding

type GenerateInput struct {
	Text string
}

type GenerateOutput struct {
	Vector []float32
}

type GenerateManyInput struct {
	Texts []string
}

type GenerateManyOutput struct {
	Vectors [][]float32
}
