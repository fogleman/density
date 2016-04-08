package density

import "math"

type KernelItem struct {
	Dx, Dy int
	Weight float64
}

type Kernel []KernelItem

func NewKernel(n int) Kernel {
	var result Kernel
	for dy := -n; dy <= n; dy++ {
		for dx := -n; dx <= n; dx++ {
			d := math.Sqrt(float64(dx*dx + dy*dy))
			w := math.Max(0, 1-d/float64(n))
			w = math.Pow(w, 2)
			result = append(result, KernelItem{dx, dy, w})
		}
	}
	return result
}
