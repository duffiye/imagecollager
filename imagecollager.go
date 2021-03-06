package main

import (
	"image"
	"image/color"
	"image/draw"
	"log"
	"math"
	"os"
	"sort"
	"strconv"

	"github.com/fogleman/imview"
	"github.com/nfnt/resize"
)

func Width(i image.Image) int {
	return i.Bounds().Max.X - i.Bounds().Min.X
}

func Height(i image.Image) int {
	return i.Bounds().Max.Y - i.Bounds().Min.Y
}

type MyImage struct {
	value *image.RGBA
}

func (i *MyImage) Set(x, y int, c color.Color) {
	i.value.Set(x, y, c)
}

func (i *MyImage) ColorModel() color.Model {
	return i.value.ColorModel()
}

func (i *MyImage) Bounds() image.Rectangle {
	return i.value.Bounds()
}

func (i *MyImage) At(x, y int) color.Color {
	return i.value.At(x, y)
}

type Circle struct {
	p image.Point
	r int
}

func (c *Circle) ColorModel() color.Model {
	return color.AlphaModel
}

func (c *Circle) Bounds() image.Rectangle {
	return image.Rect(c.p.X-int(c.r), c.p.Y-int(c.r), c.p.X+int(c.r), c.p.Y+int(c.r))
}

func (c *Circle) At(x, y int) color.Color {
	xx, yy, rr := float64(x-c.p.X)+0.5, float64(y-c.p.Y)+0.5, float64(c.r)
	if xx*xx+yy*yy < rr*rr {
		return color.Alpha{255}
	}
	return color.Alpha{0}
}

type Size struct {
	width  uint
	height uint
}

type ImageShape string

const (
	RectangleShape ImageShape = "Rectangle"
	CircleShape    ImageShape = "Circle"
	CircleDiameter            = 0.8
)

func drawLine(img *image.RGBA, line_width int, space_from_end_x int, space_from_end_y int) {
	for i := img.Bounds().Max.X - line_width - space_from_end_x; i < img.Bounds().Max.X-space_from_end_x; i++ {
		img.Set(i, img.Bounds().Max.Y-space_from_end_y, color.RGBA{255, 255, 255, 255})
	}
}

func (bgImg *MyImage) drawRaw(innerImg image.Image, sp image.Point, width uint, height uint) {
	resizedImg := resize.Resize(width, height, innerImg, resize.Lanczos3)
	w := int(Width(resizedImg))
	h := int(Height(resizedImg))
	draw.Draw(bgImg, image.Rectangle{sp, image.Point{sp.X + w, sp.Y + h}}, resizedImg, image.ZP, draw.Src)
}

func (bgImg *MyImage) drawInCircle(innerImg image.Image, sp image.Point, width uint, height uint, diameter int) {
	resizedImg := resize.Resize(width, height, innerImg, resize.Lanczos3)

	r := diameter
	if r > Width(resizedImg) {
		r = Width(resizedImg)
	}

	if r > Height(resizedImg) {
		r = int(Height(resizedImg))
	}

	mask := &Circle{image.Point{Width(resizedImg) / 2, Height(resizedImg) / 2}, r / 2}

	draw.DrawMask(bgImg, image.Rectangle{sp, image.Point{sp.X + Width(resizedImg), sp.Y + Height(resizedImg)}}, resizedImg, image.ZP, mask, image.ZP, draw.Over)
}

func makeImageCollage(desiredWidth int, desiredHeight int, numberOfRows int, shape ImageShape, images ...image.Image) *MyImage {

	sort.Slice(images, func(i, j int) bool {
		return Height(images[i]) > Height(images[j])
	})

	numberOfColumns := len(images) / numberOfRows
	imagesMatrix := make([][]image.Image, numberOfRows)

	currentIndex := 0
	maxNumberOfColumns := 0
	for idx := 0; idx < numberOfRows; idx++ {
		columnsInRow := numberOfColumns
		if len(images)%numberOfRows > 0 && (numberOfRows-idx)*numberOfColumns < len(images)-currentIndex {
			columnsInRow++
		}

		if columnsInRow > maxNumberOfColumns {
			maxNumberOfColumns = columnsInRow
		}

		imagesMatrix[idx] = images[currentIndex : currentIndex+columnsInRow]
		currentIndex += columnsInRow
	}

	maxWidth := uint(0)
	imagesSize := make([][]Size, numberOfRows)
	for row := 0; row < numberOfRows; row++ {
		imagesSize[row] = make([]Size, len(imagesMatrix[row]))

		calculatedWidth := math.Floor(float64(desiredWidth) / float64(len(imagesMatrix[row])))

		rowWidth := uint(0)
		rowHeight := uint(0)
		for col := 0; col < len(imagesMatrix[row]); col++ {
			originalWidth := float64(Width(imagesMatrix[row][col]))
			originalHeight := float64(Height(imagesMatrix[row][col]))
			resizeFactor := calculatedWidth / originalWidth

			w := uint(originalWidth * resizeFactor)
			h := uint(originalHeight * resizeFactor)
			imagesSize[row][col] = Size{w, h}

			if shape == RectangleShape {
				rowWidth += w
			} else {
				rowWidth += uint(math.Min(float64(w), float64(h)) * CircleDiameter)
			}
			rowHeight += h

		}

		if rowWidth > maxWidth {
			maxWidth = rowWidth
		}
	}

	maxHeight := uint(0)
	for col := 0; col < maxNumberOfColumns; col++ {
		colHeight := uint(0)
		for row := 0; row < numberOfRows; row++ {
			if len(imagesSize[row]) > col {
				if shape == RectangleShape {
					colHeight += imagesSize[row][col].height
				} else {
					colHeight += uint(math.Min(float64(imagesSize[row][col].height), float64(imagesSize[row][col].width)) * CircleDiameter)
				}
			}
		}

		if colHeight > maxHeight {
			maxHeight = colHeight
		}
	}

	padding := 1

	if shape == CircleShape {
		padding = 20
	}

	rectangleEnd := image.Point{int(maxWidth) + (maxNumberOfColumns-1)*padding + 2*padding, int(maxHeight) + (numberOfRows-1)*padding + 2*padding}

	output := MyImage{image.NewRGBA(image.Rectangle{image.ZP, rectangleEnd})}

	sp_x, sp_y := 0, 0
	for row := 0; row < numberOfRows; row++ {
		rowHeight := uint(0)

		calculatedWidth := math.Floor(float64(desiredWidth) / float64(len(imagesMatrix[row])))
		for col := 0; col < len(imagesMatrix[row]); col++ {
			resizeFactor := float64(1)
			originalWidth := float64(Width(imagesMatrix[row][col]))
			resizeFactor = calculatedWidth / originalWidth

			w := uint(originalWidth * resizeFactor)
			h := uint(float64(Height(imagesMatrix[row][col])) * resizeFactor)

			if col == 0 {
				sp_x = padding
			}

			if row == 0 {
				sp_y = padding
			}

			sp := image.Point{sp_x, sp_y}

			if shape == RectangleShape {
				output.drawRaw(imagesMatrix[row][col], sp, w, h)
			} else {
				w = uint(math.Min(float64(w), float64(h)) * CircleDiameter)
				h = w

				output.drawInCircle(imagesMatrix[row][col], sp, w, h, int(w))
			}

			sp_x += int(w) + padding

			if h > rowHeight {
				rowHeight = h
			}

		}

		sp_x = 0
		sp_y += int(rowHeight) + padding

	}

	return &output
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("No shape or number of rows defined")
	} else {
		imageShape := ImageShape(os.Args[1])
		numberOfRows, errNr := strconv.Atoi(os.Args[2])

		if errNr == nil && (imageShape == RectangleShape || imageShape == CircleShape) {
			images := make([]image.Image, len(os.Args)-3)

			for i := 3; i < len(os.Args); i++ {
				fimg, _ := os.Open(os.Args[i])
				defer fimg.Close()
				img, _, _ := image.Decode(fimg)

				images[i-3] = img
			}

			output := makeImageCollage(800, 800, numberOfRows, imageShape, images...)
			imview.Show(output.value)
		} else {
			log.Fatal("No shape or number of rows defined")
		}
	}

	// output := MyImage{image.NewRGBA(image.Rectangle{image.ZP, image.Point{400, 400}})}

	// fimg, _ := os.Open("dog.jpg")
	// defer fimg.Close()
	// img, _, _ := image.Decode(fimg)
	// output.drawRaw(img, image.Point{100, 100}, 180, 150)
	// // output.drawInCircle(img, image.Point{100, 100}, 180, 180, 150)

	// imview.Show(output.value)

}
