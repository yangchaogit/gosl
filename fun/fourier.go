// Copyright 2016 The Gosl Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fun

import (
	"math"
	"math/cmplx"

	"github.com/cpmech/gosl/chk"
	"github.com/cpmech/gosl/io"
	"github.com/cpmech/gosl/plt"
	"github.com/cpmech/gosl/utl"
)

// Smoothing kinds
var (

	// No smoothing
	SmoNoneKind = io.NewEnum("None", "fun.smoothing", "L", "No smoothing")

	// Lanczos (sinc) smoothing kind
	SmoLanczosKind = io.NewEnum("Lanczos", "fun.smoothing", "L", "Lanczos (sinc) smoothing kind")

	// Cesaro
	SmoCesaroKind = io.NewEnum("Cesaro", "fun.smoothing", "L", "Lanczos (sinc) smoothing kind")

	// Raised Cosine
	SmoRcosKind = io.NewEnum("Rcos", "fun.smoothing", "L", "Lanczos (sinc) smoothing kind")
)

// FourierInterp performs interpolation using truncated Fourier series
//
//                  N/2 - 1
//                   ————          +i k X[j]
//        f(x[j]) =  \     A[k] ⋅ e              with    X[j] = 2 π j / N
//                   /
//                   ————
//                  k = -N/2
//
//   x ϵ [0, 2π]
//
//   Equation (2.1.27) of [1]. Note that f=u in [1] and A[k] is the tilde(u[k]) of [1]
//
//   Reference:
//     [1] Canuto C, Hussaini MY, Quarteroni A, Zang TA (2006) Spectral Methods: Fundamentals in
//         Single Domains. Springer. 563p
//
type FourierInterp struct {
	N int          // number of terms. must be power of 2; i.e. N = 2ⁿ
	A []complex128 // coefficients for interpolation. from FFT
	S []complex128 // smothing coefficients
}

// NewFourierInterp allocates a new FourierInterp object
//
//                    N - 1
//                1   ————             -i k X[j]
//        A[k] = ———  \     f(x[j]) ⋅ e              with    X[j] = 2 π j / N
//                N   /
//                    ————
//                   j = 0
//
//  INPUT:
//    N -- number of terms. must be power of 2; i.e. N = 2ⁿ
//
//    smoothing -- type of smoothing: use SmoNoneKind for no smoothing
//
//  NOTE: (1) x ϵ [0, 2π]
//        (2) remember to call CalcA to calculate coefficients A!
//
//   Equation (2.1.25) of [1]. Note that f=u in [1] and A[k] is the tilde(u[k]) of [1]
//
//   Reference:
//     [1] Canuto C, Hussaini MY, Quarteroni A, Zang TA (2006) Spectral Methods: Fundamentals in
//         Single Domains. Springer. 563p
//
func NewFourierInterp(N int, smoothing io.Enum) (o *FourierInterp, err error) {

	// check
	if N%2 != 0 {
		err = chk.Err("N must be even. N=%d is invalid\n", N)
		return
	}

	// allocate
	o = new(FourierInterp)
	o.N = N
	o.A = make([]complex128, o.N)
	o.S = make([]complex128, o.N)

	// compute smoothing coefficients
	n := float64(o.N)
	σ := func(k float64) float64 { return 1.0 }
	switch smoothing {
	case SmoLanczosKind:
		σ = func(k float64) float64 { return Sinc(2 * k * π / n) }
	case SmoRcosKind:
		σ = func(k float64) float64 { return (1.0 + math.Cos(2*k*π/n)) / 2.0 }
	case SmoCesaroKind:
		σ = func(k float64) float64 { return 1.0 - math.Abs(k)/(1.0+n/2.0) }
	}

	for j := 0; j < o.N; j++ {
		o.S[j] = complex(σ(o.K(j)), 0)
	}
	return
}

// CalcA calculates the coefficients A of the interpolation ousing FFT
func (o *FourierInterp) CalcA(f Ss) (err error) {

	// compute f(x[j]) and set A[j] with f(x[j]) / N
	var fxj float64
	n := float64(o.N)
	for j := 0; j < o.N; j++ {
		xj := 2.0 * math.Pi * float64(j) / n
		fxj, err = f(xj)
		if err != nil {
			return
		}
		o.A[j] = complex(fxj/n, 0)
	}

	// perform Fourier transform to find A[j]
	err = Dft1d(o.A, false)
	return
}

// K computes k-index from j-index where j corresponds to the FFT index
//
//   FFT returns the A coefficients as:
//
//      {A[0], A[1], ..., A[N/2-1], A[-N/2], A[-N/2+1], ... A[-1]}
//
//   k ϵ [-N/2, N/2-1]
//   j ϵ [0, N-1]
//
//   Example with N = 8:
//
//        j=0 ⇒ k=0      j=4 ⇒ k=-4
//        j=1 ⇒ k=1      j=5 ⇒ k=-3
//        j=2 ⇒ k=2      j=6 ⇒ k=-2
//        j=3 ⇒ k=3      j=7 ⇒ k=-1
//
func (o *FourierInterp) K(j int) float64 {
	h := o.N / 2
	k := j - (j/h)*o.N
	return float64(k)
}

// I computes the interpolation
//
//                  N/2 - 1
//                    ————          +i k x
//        I {f}(x) =  \     A[k] ⋅ e
//         N          /
//                    ————
//                   k = -N/2
//
//   x ϵ [0, 2π]
//
//   Equation (2.1.28) of [1]. Note that f=u in [1] and A[k] is the tilde(u[k]) of [1]
//
//   Reference:
//     [1] Canuto C, Hussaini MY, Quarteroni A, Zang TA (2006) Spectral Methods: Fundamentals in
//         Single Domains. Springer. 563p
//
//  NOTE: remember to call CalcA to calculate coefficients A!
//
func (o *FourierInterp) I(x float64) float64 {
	var res complex128
	for j := 0; j < o.N; j++ {
		res += o.S[j] * o.A[j] * cmplx.Exp(complex(0, o.K(j)*x))
	}
	return real(res)
}

// DI computes the p-derivative of the interpolation
//
//                   p       N/2 - 1
//        p         d(I{f})    ————       p           +i k x
//       DI{f}(x) = ——————— =  \     (i⋅k)  ⋅ A[k] ⋅ e
//        N             p      /
//                    dx       ————
//                            k = -N/2
//
//   x ϵ [0, 2π]
//
//  NOTE: remember to call CalcA to calculate coefficients A!
//
func (o *FourierInterp) DI(p int, x float64) float64 {
	var res complex128
	pc := complex(float64(p), 0)
	for j := 0; j < o.N; j++ {
		ik := complex(0, o.K(j))
		ikp := cmplx.Pow(ik, pc)
		res += ikp * o.S[j] * o.A[j] * cmplx.Exp(complex(0, o.K(j)*x))
	}
	return real(res)
}

// Plot plots interpolated curve
//   option -- 1: plot only f(x)
//             2: plot both f(x) and df/dx(x)
//             3: plot all f(x), df/dx(x) and d^2f/dx^2
//             4: plot only df/dx(x)
//             5: plot only d^2f/dx^2(x)
//             6: plot df^p/dx^p
//   p      -- order of the derivative to plot if option == 6
//   dfdx   -- is the analytic df/dx(x) [optional]
//   d2fdx2 -- is the analytic d^2f/dx^2(x) [optional]
func (o *FourierInterp) Plot(option, p int, f, dfdx, d2fdx2 Ss, argsF, argsI, argsD1, argsD2 *plt.A) {

	// set arguments
	if argsF == nil {
		argsF = &plt.A{L: "f(x)", C: plt.C(0, 1), NoClip: true}
	}
	if argsI == nil {
		argsI = &plt.A{L: "I{f}(x)", C: plt.C(1, 1), NoClip: true}
	}
	if argsD1 == nil {
		argsD1 = &plt.A{L: "D1I{f}(x)", C: plt.C(2, 1), NoClip: true}
	}
	if argsD2 == nil {
		argsD2 = &plt.A{L: "D2I{f}(x)", C: plt.C(3, 1), NoClip: true}
	}

	// graph points
	npts := 2001
	xx := utl.LinSpace(0, 2.0*math.Pi, npts)

	// options
	withF := option == 1 || option == 2 || option == 3
	firstD := option == 2 || option == 3 || option == 4
	secondD := option == 3 || option == 5

	// allocate arrays
	var y1, y2 []float64
	if withF {
		y1 = make([]float64, npts)
		y2 = make([]float64, npts)
	}
	var y3, y4 []float64
	if firstD {
		y3 = make([]float64, npts)
		y4 = make([]float64, npts)
	}
	var y5, y6 []float64
	if secondD {
		y5 = make([]float64, npts)
		y6 = make([]float64, npts)
	}
	var y7 []float64
	if option == 6 {
		y7 = make([]float64, npts)
	}

	// compute values
	for i := 0; i < npts; i++ {
		x := xx[i]
		if withF {
			if f != nil {
				fx, err := f(x)
				if err != nil {
					chk.Panic("f(x) failed:\n%v\n", err)
				}
				y1[i] = fx
			}
			y2[i] = o.I(x)
		}
		if firstD {
			if dfdx != nil {
				dfx, err := dfdx(x)
				if err != nil {
					chk.Panic("df/dx(x) failed:\n%v\n", err)
				}
				y3[i] = dfx
			}
			y4[i] = o.DI(1, x)
		}
		if secondD {
			if d2fdx2 != nil {
				ddfx, err := d2fdx2(x)
				if err != nil {
					chk.Panic("d2f/dx2(x) failed:\n%v\n", err)
				}
				y5[i] = ddfx
			}
			y6[i] = o.DI(2, x)
		}
		if option == 6 {
			y7[i] = o.DI(p, x)
		}
	}

	// plot
	if option == 2 {
		plt.Subplot(2, 1, 1)
	}
	if option == 3 {
		plt.Subplot(3, 1, 1)
	}
	if withF {
		if f != nil {
			plt.Plot(xx, y1, argsF)
		}
		plt.Plot(xx, y2, argsI)
		plt.HideTRborders()
		plt.Gll("$x$", "$f(x)$", nil)
	}
	if option == 2 {
		plt.Subplot(2, 1, 2)
	}
	if option == 3 {
		plt.Subplot(3, 1, 2)
	}
	if firstD {
		argsF.L = "dfdx"
		//plt.Plot(X, yX, argsX)
		if dfdx != nil {
			plt.Plot(xx, y3, argsF)
		}
		plt.Plot(xx, y4, argsD1)
		plt.HideTRborders()
		plt.Gll("$x$", "$\\frac{\\mathrm{d}f(x)}{\\mathrm{d}x}$", nil)
	}
	if option == 3 {
		plt.Subplot(3, 1, 3)
	}
	if secondD {
		argsF.L = "d2fdx2"
		if d2fdx2 != nil {
			plt.Plot(xx, y5, argsF)
		}
		plt.Plot(xx, y6, argsD2)
		plt.HideTRborders()
		plt.Gll("$x$", "$\\frac{\\mathrm{d}^2f(x)}{\\mathrm{d}x^2}$", nil)
	}
	if option == 6 {
		argsI.L = io.Sf("D%d", p)
		plt.Plot(xx, y7, argsI)
		plt.HideTRborders()
		plt.Gll("$x$", io.Sf("$\\frac{\\mathrm{d}^{%d}f(x)}{\\mathrm{d}x^{%d}}$", p, p), nil)
	}
}
