package mandel

// var BasicParams = Tile{
// 	Region: SeahorseValley,
// }

// Region within the Mandelbrot set
type Region struct {
	Xmin, Xmax float64
	Ymin, Ymax float64
}

// Classic regions / landmarks in the Mandelbrot set
var (
	// Seahorse Valley – dense filaments and repeating “seahorse” curls
	SeahorseValley = Region{
		Xmin: -0.8,
		Xmax: -0.7,
		Ymin: 0.05,
		Ymax: 0.15,
	}

	// Elephant Valley – large bulb with trunk-like tendrils
	ElephantValley = Region{
		Xmin: -1.85,
		Xmax: -1.75,
		Ymin: -0.10,
		Ymax: -0.02,
	}

	// Spiral Minibrot – small Mandelbrot copy with tight spiral arms
	SpiralMinibrot = Region{
		Xmin: -0.7435,
		Xmax: -0.7420,
		Ymin: 0.1310,
		Ymax: 0.1325,
	}

	// Triple Spiral – threefold symmetric spiral structure
	TripleSpiral = Region{
		Xmin: -0.7480,
		Xmax: -0.7450,
		Ymin: 0.0950,
		Ymax: 0.0980,
	}

	// Valley of the Dragon – deep, highly detailed spiral filaments
	ValleyOfTheDragon = Region{
		Xmin: -0.7400,
		Xmax: -0.7350,
		Ymin: 0.1800,
		Ymax: 0.1850,
	}

	// Minibrot in a Mini-Spiral – self-similar Mandelbrot copy inside a spiral arm
	MinibrotInMiniSpiral = Region{
		Xmin: -1.7390,
		Xmax: -1.7375,
		Ymin: -0.0235,
		Ymax: -0.0220,
	}
)

type Tile struct {
	X0, Y0 int // top-left pixel in global image
	W, H   int // tile width & height
}
