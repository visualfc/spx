package matrix

import (
	"math"
)

/*
 | a | c | tx|
 | b | d | ty|
 | 0 | 0 | 1 |
*/

type Matrix struct {
	A, B, C, D, TX, TY float64
}

func NewMatrix() *Matrix {
	return &Matrix{
		1, 0,
		0, 1,
		0, 0,
	}
}

func (m *Matrix) FromArray(array []float64) {
	m.A = array[0]
	m.B = array[1]
	m.C = array[2]
	m.D = array[3]
	m.TX = array[4]
	m.TY = array[5]
}

func (m *Matrix) ToTransposeArray() []float64 {
	return []float64{
		m.A, m.B, 0,
		m.C, m.D, 0,
		m.TX, m.TY, 1,
	}
}

func (m *Matrix) Apply(x, y float64) (float64, float64) {
	rx := m.A*x + m.C*y + m.TX
	ry := m.B*x + m.D*y + m.TY
	return rx, ry
}

func (m *Matrix) ApplyInverse(x, y float64) (float64, float64) {
	id := 1.0 / (m.A*m.D + m.C*-m.B)
	rx := id * ((m.D * x) + (-m.C * y) + ((m.TY * m.C) - (m.TX * m.D)))
	ry := id * ((m.A * y) + (-m.B * x) + ((-m.TY * m.A) + (m.TX * m.B)))
	return rx, ry
}

func Identity() Matrix {
	return Matrix{
		1, 0,
		0, 1,
		0, 0,
	}
}

func TranslateMatrix(tx float64, ty float64) Matrix {
	return Matrix{
		1, 0,
		0, 1,
		tx, ty,
	}
}

func MatrixFromTransform(x, y, pivotX, pivotY, scaleX, scaleY, rotation, skewX, skewY float64) Matrix {
	if scaleX == 0 {
		scaleX = 1
	}
	if scaleY == 0 {
		scaleY = 1
	}
	a := math.Cos(rotation+skewY) * scaleX
	b := math.Sin(rotation+skewY) * scaleX
	c := -math.Sin(rotation-skewX) * scaleY
	d := math.Cos(rotation-skewX) * scaleY
	return Matrix{
		a,
		b,
		c,
		d,
		x - (pivotX*a + pivotY*c),
		y - (pivotX*b + pivotY*d),
	}
}

func (m *Matrix) SetMatrix(a, b, c, d, tx, ty float64) {
	m.A = a
	m.B = b
	m.C = c
	m.D = d
	m.TX = tx
	m.TY = ty
}

func (m *Matrix) SetTransform(x, y, scaleX, scaleY, rotation, skewX, skewY, pivotX, pivotY float64) {
	if scaleX == 0 {
		scaleX = 1
	}
	if scaleY == 0 {
		scaleY = 1
	}
	m.A = math.Cos(rotation+skewY) * scaleX
	m.B = math.Sin(rotation+skewY) * scaleX
	m.C = -math.Sin(rotation-skewX) * scaleY
	m.D = math.Cos(rotation-skewX) * scaleY

	m.TX = x - (pivotX*m.A + pivotY*m.C)
	m.TY = y - (pivotX*m.B + pivotY*m.D)
}

func (m *Matrix) Matrix() (a, b, c, d, tx, ty float64) {
	return m.A, m.B, m.C, m.D, m.TX, m.TY
}

func (m *Matrix) Scale(sx, sy float64) {
	m.A *= sx
	m.B *= sy
	m.C *= sx
	m.D *= sy
	m.TX *= sx
	m.TY *= sy
}

func (m *Matrix) Translate(x, y float64) {
	m.TX += x
	m.TY += y
}

func (m *Matrix) Rotate(angle float64) {
	cos := math.Cos(angle)
	sin := math.Sin(angle)
	a1 := m.A
	c1 := m.C
	tx1 := m.TX

	m.A = a1*cos - m.B*sin
	m.B = a1*sin + m.B*cos
	m.C = c1*cos - m.D*sin
	m.D = c1*sin + m.D*cos
	m.TX = tx1*cos - m.TY*sin
	m.TY = tx1*sin + m.TY*cos
}

func (m *Matrix) Append(tr *Matrix) *Matrix {
	a1 := m.A
	b1 := m.B
	c1 := m.C
	d1 := m.D

	m.A = tr.A*a1 + tr.B*c1
	m.B = tr.A*b1 + tr.B*d1
	m.C = tr.C*a1 + tr.D*c1
	m.D = tr.C*b1 + tr.D*d1

	m.TX = tr.TX*a1 + tr.TY*c1 + m.TX
	m.TY = tr.TX*b1 + tr.TY*d1 + m.TY
	return m
}

func (m *Matrix) Invert() {
	a1 := m.A
	b1 := m.B
	c1 := m.C
	d1 := m.D
	tx1 := m.TX
	n := a1*d1 - b1*c1

	m.A = d1 / n
	m.B = -b1 / n
	m.C = -c1 / n
	m.D = a1 / n
	m.TX = (c1*m.TY - d1*tx1) / n
	m.TY = -(a1*m.TY - b1*tx1) / n
}

func (m Matrix) TransformPoint(x, y float64) (tx, ty float64) {
	tx = x*m.A + y*m.C + m.TX
	ty = x*m.B + y*m.D + m.TY
	return
}

func (m Matrix) TransformPoints(points []float64) (pt []float64) {
	sz := len(points)
	pt = make([]float64, sz)
	for i, j := 0, 1; j < sz; i, j = i+2, j+2 {
		x := points[i]
		y := points[j]
		pt[i] = x*m.A + y*m.C + m.TX
		pt[j] = x*m.B + y*m.D + m.TY
	}
	return pt
}

func (m Matrix) TransformPoints32(points []float64, out []float32) {
	for i, j := 0, 1; j < len(points); i, j = i+2, j+2 {
		x := points[i]
		y := points[j]
		out[i] = float32(x*m.A + y*m.C + m.TX)
		out[j] = float32(x*m.B + y*m.D + m.TY)
	}
}
