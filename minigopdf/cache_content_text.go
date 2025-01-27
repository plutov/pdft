package gopdf

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
)

//ContentTypeCell cell
const ContentTypeCell = 0

//ContentTypeText text
const ContentTypeText = 1

type cacheContentText struct {
	//---setup---
	rectangle      *Rect
	textColor      Rgb
	grayFill       float64
	fontCountIndex int //Curr.Font_FontCount+1
	fontSize       int
	fontStyle      int
	setXCount      int //จำนวนครั้งที่ใช้ setX
	x, y           float64
	fontSubset     *SubsetFontObj
	pageheight     float64
	contentType    int
	cellOpt        CellOption
	lineWidth      float64
	//---result---
	content                            bytes.Buffer
	text                               bytes.Buffer
	cellWidthPdfUnit, textWidthPdfUnit float64
	cellHeightPdfUnit                  float64
}

func (c *cacheContentText) isSame(cache cacheContentText) bool {
	if c.rectangle != nil {
		//if rectangle != nil we assumes this is not same content
		return false
	}
	if c.textColor.equal(cache.textColor) &&
		c.grayFill == cache.grayFill &&
		c.fontCountIndex == cache.fontCountIndex &&
		c.fontSize == cache.fontSize &&
		c.fontStyle == cache.fontStyle &&
		c.setXCount == cache.setXCount &&
		c.y == cache.y {
		return true
	}

	return false
}

func (c *cacheContentText) setPageHeight(pageheight float64) {
	c.pageheight = pageheight
}

func (c *cacheContentText) pageHeight() float64 {
	return c.pageheight //841.89
}

func convertTypoUnit(val float64, unitsPerEm uint, fontSize float64) float64 {
	val = val * 1000.00 / float64(unitsPerEm)
	return val * fontSize / 1000.0
}

func (c *cacheContentText) calTypoAscender() float64 {
	return convertTypoUnit(float64(c.fontSubset.ttfp.TypoAscender()), c.fontSubset.ttfp.UnitsPerEm(), float64(c.fontSize))
}

func (c *cacheContentText) calTypoDescender() float64 {
	return convertTypoUnit(float64(c.fontSubset.ttfp.TypoDescender()), c.fontSubset.ttfp.UnitsPerEm(), float64(c.fontSize))
}

func (c *cacheContentText) calY() (float64, error) {
	pageHeight := c.pageHeight()
	if c.contentType == ContentTypeText {
		return pageHeight - c.y, nil
	} else if c.contentType == ContentTypeCell {
		y := float64(0.0)
		if c.cellOpt.Align&Bottom == Bottom {
			y = pageHeight - c.y - c.cellHeightPdfUnit - c.calTypoDescender()
		} else if c.cellOpt.Align&Middle == Middle {
			y = pageHeight - c.y - c.cellHeightPdfUnit*0.5 - (c.calTypoDescender()+c.calTypoAscender())*0.5
		} else {
			//top
			y = pageHeight - c.y - c.calTypoAscender()
		}

		return y, nil
	}
	return 0.0, errors.New("contentType not found")
}

func (c *cacheContentText) calX() (float64, error) {
	if c.contentType == ContentTypeText {
		return c.x, nil
	} else if c.contentType == ContentTypeCell {
		x := float64(0.0)
		if c.cellOpt.Align&Right == Right {
			x = c.x + c.cellWidthPdfUnit - c.textWidthPdfUnit
		} else if c.cellOpt.Align&Center == Center {
			x = c.x + c.cellWidthPdfUnit*0.5 - c.textWidthPdfUnit*0.5
		} else {
			x = c.x
		}
		return x, nil
	}
	return 0.0, errors.New("contentType not found")
}

func (c *cacheContentText) toStream(protection *PDFProtection) (*bytes.Buffer, error) {

	var stream bytes.Buffer
	r := c.textColor.r
	g := c.textColor.g
	b := c.textColor.b
	x, err := c.calX()
	if err != nil {
		return nil, err
	}
	y, err := c.calY()
	if err != nil {
		return nil, err
	}

	stream.WriteString("BT\n")
	stream.WriteString(fmt.Sprintf("%0.2f %0.2f TD\n", x, y))
	stream.WriteString("/" + FontKeyword + strconv.Itoa(c.fontCountIndex) + " " + strconv.Itoa(c.fontSize) + " Tf\n")
	if r+g+b != 0 {
		rFloat := float64(r) * 0.00392156862745
		gFloat := float64(g) * 0.00392156862745
		bFloat := float64(b) * 0.00392156862745
		rgb := fmt.Sprintf("%0.2f %0.2f %0.2f rg\n", rFloat, gFloat, bFloat)
		stream.WriteString(rgb)
	} else {
		//c.AppendStreamSetGrayFill(grayFill)
	}

	//stream.WriteString("[<" + c.content.String() + ">] TJ\n")
	stream.WriteString(c.content.String())
	stream.WriteString("ET\n")

	if c.fontStyle&Underline == Underline {
		underlineStream, err := c.underline(c.x, c.y, c.x+c.cellWidthPdfUnit, c.y)
		if err != nil {
			return nil, err
		}
		_, err = underlineStream.WriteTo(&stream)
		if err != nil {
			return nil, err
		}
	}

	c.drawBorder(&stream)

	return &stream, nil
}

func (c *cacheContentText) drawBorder(stream *bytes.Buffer) error {

	//stream.WriteString(fmt.Sprintf("%.2f w\n", 0.1))
	lineOffset := c.lineWidth * 0.5

	if c.cellOpt.Border&Top == Top {

		startX := c.x - lineOffset
		startY := c.pageHeight() - c.y
		endX := c.x + c.cellWidthPdfUnit + lineOffset
		endY := startY
		_, err := stream.WriteString(fmt.Sprintf("%0.2f %0.2f m %0.2f %0.2f l s\n", startX, startY, endX, endY))
		if err != nil {
			return err
		}
	}

	if c.cellOpt.Border&Left == Left {
		startX := c.x
		startY := c.pageHeight() - c.y
		endX := c.x
		endY := startY - c.cellHeightPdfUnit
		_, err := stream.WriteString(fmt.Sprintf("%0.2f %0.2f m %0.2f %0.2f l s\n", startX, startY, endX, endY))
		if err != nil {
			return err
		}
	}

	if c.cellOpt.Border&Right == Right {
		startX := c.x + c.cellWidthPdfUnit
		startY := c.pageHeight() - c.y
		endX := c.x + c.cellWidthPdfUnit
		endY := startY - c.cellHeightPdfUnit
		_, err := stream.WriteString(fmt.Sprintf("%0.2f %0.2f m %0.2f %0.2f l s\n", startX, startY, endX, endY))
		if err != nil {
			return err
		}
	}

	if c.cellOpt.Border&Bottom == Bottom {
		startX := c.x - lineOffset
		startY := c.pageHeight() - c.y - c.cellHeightPdfUnit
		endX := c.x + c.cellWidthPdfUnit + lineOffset
		endY := startY
		_, err := stream.WriteString(fmt.Sprintf("%0.2f %0.2f m %0.2f %0.2f l s\n", startX, startY, endX, endY))
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *cacheContentText) underline(startX float64, startY float64, endX float64, endY float64) (*bytes.Buffer, error) {

	if c.fontSubset == nil {
		return nil, errors.New("error AppendUnderline not found font")
	}
	unitsPerEm := float64(c.fontSubset.ttfp.UnitsPerEm())
	h := c.pageHeight()
	ut := float64(c.fontSubset.GetUt())
	up := float64(c.fontSubset.GetUp())
	var buff bytes.Buffer
	textH := ContentObj_CalTextHeight(c.fontSize)
	arg3 := float64(h) - (float64(startY) - ((up / unitsPerEm) * float64(c.fontSize))) - textH
	arg4 := (ut / unitsPerEm) * float64(c.fontSize)
	buff.WriteString(fmt.Sprintf("%0.2f %0.2f %0.2f -%0.2f re f\n", startX, arg3, endX-startX, arg4))
	//fmt.Printf("arg3=%f arg4=%f\n", arg3, arg4)

	return &buff, nil
}

func (c *cacheContentText) createContent() (float64, float64, error) {

	c.content.Truncate(0) //clear
	cellWidthPdfUnit, cellHeightPdfUnit, textWidthPdfUnit, err := createContent(c.fontSubset, c.text.String(), c.fontSize, c.rectangle, &c.content)
	if err != nil {
		return 0, 0, err
	}
	c.cellWidthPdfUnit = cellWidthPdfUnit
	c.cellHeightPdfUnit = cellHeightPdfUnit
	c.textWidthPdfUnit = textWidthPdfUnit
	return cellWidthPdfUnit, cellHeightPdfUnit, nil
}

func createContent(f *SubsetFontObj, text string, fontSize int, rectangle *Rect, out *bytes.Buffer) (float64, float64, float64, error) {

	unitsPerEm := int(f.ttfp.UnitsPerEm())
	var leftRune rune
	var leftRuneIndex uint
	sumWidth := int(0)
	//fmt.Printf("unitsPerEm = %d", unitsPerEm)
	out.WriteString("[<")
	runeIndex := 0
	for i, r := range text {

		glyphindex, err := f.CharIndex(r)
		if err != nil {
			return 0, 0, 0, err
		}

		pairvalPdfUnit := 0
		if i > 0 && f.ttfFontOption.UseKerning { //kerning
			pairval := kern(f, leftRune, r, leftRuneIndex, glyphindex)
			pairvalPdfUnit = convertTTFUnit2PDFUnit(int(pairval), unitsPerEm)
			if pairvalPdfUnit != 0 && out != nil {
				out.WriteString(fmt.Sprintf(">%d<", (-1)*pairvalPdfUnit))
			}
		}

		if f.funcTextriseOverride != nil {
			//fmt.Printf("gopdf i=%d\n", runeIndex)
			val := textrise(f, leftRune, r, leftRuneIndex, glyphindex, fontSize, text, runeIndex)
			out.WriteString(">] TJ\n")
			out.WriteString(fmt.Sprintf("%.2f Ts\n", val))
			out.WriteString("[<")
		}

		if out != nil {
			out.WriteString(fmt.Sprintf("%04X", glyphindex))
		}
		width, err := f.CharWidth(r)
		if err != nil {
			return 0, 0, 0, err
		}

		sumWidth += int(width) + int(pairvalPdfUnit)
		leftRune = r
		leftRuneIndex = glyphindex
		runeIndex++
	}
	out.WriteString(">] TJ\n")

	cellWidthPdfUnit := float64(0)
	cellHeightPdfUnit := float64(0)
	if rectangle == nil {
		cellWidthPdfUnit = float64(sumWidth) * (float64(fontSize) / 1000.0)
		typoAscender := convertTypoUnit(float64(f.ttfp.TypoAscender()), f.ttfp.UnitsPerEm(), float64(fontSize))
		typoDescender := convertTypoUnit(float64(f.ttfp.TypoDescender()), f.ttfp.UnitsPerEm(), float64(fontSize))
		cellHeightPdfUnit = typoAscender - typoDescender
	} else {
		cellWidthPdfUnit = rectangle.W
		cellHeightPdfUnit = rectangle.H
	}
	textWidthPdfUnit := float64(sumWidth) * (float64(fontSize) / 1000.0)

	return cellWidthPdfUnit, cellHeightPdfUnit, textWidthPdfUnit, nil
}

func kern(f *SubsetFontObj, leftRune rune, rightRune rune, leftIndex uint, rightIndex uint) int16 {

	pairVal := int16(0)
	if haveKerning, kval := f.KernValueByLeft(leftIndex); haveKerning {
		if ok, v := kval.ValueByRight(rightIndex); ok {
			pairVal = v
		}
	}

	if f.funcKernOverride != nil {
		pairVal = f.funcKernOverride(
			leftRune,
			rightRune,
			leftIndex,
			rightIndex,
			pairVal,
		)
	}
	return pairVal
}

func textrise(f *SubsetFontObj, leftRune rune, rightRune rune, leftIndex uint, rightIndex uint, fontSize int, allText string, currTextIndex int) float32 {
	pairVal := float32(0)
	if f.funcTextriseOverride != nil {
		pairVal = f.funcTextriseOverride(
			leftRune,
			rightRune,
			fontSize,
			allText,
			currTextIndex,
		)
	}
	return pairVal
}

//CacheContent Export cacheContent
type CacheContent struct {
	cacheContentText
}

//Setup setup all infomation for cacheContent
func (c *CacheContent) Setup(rectangle *Rect,
	textColor Rgb,
	grayFill float64,
	fontCountIndex int, //Curr.Font_FontCount+1
	fontSize int,
	fontStyle int,
	setXCount int, //จำนวนครั้งที่ใช้ setX
	x, y float64,
	fontSubset *SubsetFontObj,
	pageheight float64,
	contentType int,
	cellOpt CellOption,
	lineWidth float64,
) {
	c.cacheContentText = cacheContentText{
		fontSubset:     fontSubset,
		rectangle:      rectangle,
		textColor:      textColor,
		grayFill:       grayFill,
		fontCountIndex: fontCountIndex,
		fontSize:       fontSize,
		fontStyle:      fontStyle,
		setXCount:      setXCount,
		x:              x,
		y:              y,
		pageheight:     pageheight,
		contentType:    ContentTypeCell,
		cellOpt:        cellOpt,
		lineWidth:      lineWidth,
	}
}

//WriteTextToContent write text to content
func (c *CacheContent) WriteTextToContent(text string) {
	c.cacheContentText.text.WriteString(text)
}

//ToStream create stream of content
func (c *CacheContent) ToStream(protection *PDFProtection) (*bytes.Buffer, error) {
	c.cacheContentText.createContent()
	return c.cacheContentText.toStream(protection)
}
