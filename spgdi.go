package spx

import (
	"image"

	"github.com/goplus/spx/internal/gdi"
	"github.com/goplus/spx/internal/matrix"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/qiniu/x/objcache"

	spxfs "github.com/goplus/spx/fs"
)

// -------------------------------------------------------------------------------------

type drawContext struct {
	*ebiten.Image
}

type hitContext struct {
	Pos image.Point
}

type hitResult struct {
	Target interface{}
}

type Shape interface {
	draw(dc drawContext)
	hit(hc hitContext) (hr hitResult, ok bool)
}

// -------------------------------------------------------------------------------------

type sprKey struct {
	scale         float64
	direction     float64
	costume       *costume
	rotationStyle RotationStyle
}

func (p *sprKey) tryGet() *gdi.Sprite {
	if val, ok := grpSpr.TryGet(*p); ok {
		return val.(*gdi.Sprite)
	}
	return nil
}

func (p *sprKey) get(sp *Sprite) *gdi.Sprite {
	val, _ := grpSpr.Get(sp, *p)
	return val.(*gdi.Sprite)
}

func (p *sprKey) doGet(sp *Sprite) *gdi.Sprite {
	w, h := sp.g.size()
	if int(p.direction+p.costume.faceLeft)%90 == 0 {
		return p.makeSprite(0, 0, 1, w, h, sp.g.fs)
	}
	img := ebiten.NewImage(w, h)
	defer img.Dispose()
	p.drawOn(img, 0, 0, sp.g.fs)
	return gdi.NewSpriteFromScreen(img)
}

func (p *sprKey) drawOn(target *ebiten.Image, x, y float64, fs spxfs.Dir) {
	c := p.costume
	img, centerX, centerY := c.needImage(fs)

	scale := p.scale / float64(c.bitmapResolution)
	screenW, screenH := target.Size()

	op := new(ebiten.DrawImageOptions)
	geo := &op.GeoM

	direction := p.direction + c.faceLeft
	if direction == 90 {
		x = float64(screenW>>1) + x - centerX*scale
		y = float64(screenH>>1) - y - centerY*scale
		if scale != 1 {
			geo.Scale(scale, scale)
		}
		geo.Translate(x, y)
	} else {
		geo.Translate(-centerX, -centerY)
		if scale != 1 {
			geo.Scale(scale, scale)
		}
		geo.Rotate(toRadian(direction - 90))
		geo.Translate(float64(screenW>>1)+x, float64(screenH>>1)-y)
	}

	target.DrawImage(img, op)
}

func doGetSpr(ctx objcache.Context, key objcache.Key) (val objcache.Value, err error) {
	sp := ctx.(*Sprite)
	di := key.(sprKey)
	spr := di.doGet(sp)
	return spr, nil
}

var (
	grpSpr *objcache.Group = objcache.NewGroup("spr", 0, doGetSpr)
)

// -------------------------------------------------------------------------------------

type spriteDrawInfo struct {
	sprKey
	x, y    float64
	visible bool
}

func (p *spriteDrawInfo) drawOn(dc drawContext, fs spxfs.Dir) {
	sp := p.tryGet()
	if sp == nil {
		p.sprKey.drawOn(dc.Image, p.x, p.y, fs)
	} else {
		p.doDrawOn(dc, sp)
	}
}

func (p *spriteDrawInfo) draw(dc drawContext, ctx *Sprite) {
	sp := p.get(ctx)
	p.doDrawOn(dc, sp)
}

func (p *spriteDrawInfo) doDrawOn(dc drawContext, sp *gdi.Sprite) {
	img := sp.Image()
	if img.Rect.Empty() {
		return
	}
	src := ebiten.NewImageFromImage(img)
	defer src.Dispose()

	op := new(ebiten.DrawImageOptions)
	x := float64(sp.Rect.Min.X) + p.x
	y := float64(sp.Rect.Min.Y) - p.y
	op.GeoM.Translate(x, y)
	dc.DrawImage(src, op)
}

func (p *Sprite) getDrawInfo() *spriteDrawInfo {
	return &spriteDrawInfo{
		sprKey: sprKey{
			scale:         p.scale,
			direction:     p.direction,
			costume:       p.costumes[p.currentCostumeIndex],
			rotationStyle: p.rotationStyle,
		},
		x:       p.x,
		y:       p.y,
		visible: p.isVisible,
	}
}

func (p *Sprite) getGdiSprite() (spr *gdi.Sprite, pt image.Point) {
	di := p.getDrawInfo()
	if !di.visible {
		return
	}

	spr = di.get(p)
	pt = image.Pt(int(di.x), -int(di.y))
	return
}

func (p *Sprite) getTrackPos() (topx, topy int) {
	spr, pt := p.getGdiSprite()
	if spr == nil {
		return
	}

	trackp := getTrackPos(spr)
	pt = trackp.Add(pt)
	return pt.X, pt.Y
}

func (p *Sprite) draw(dc drawContext) {
	di := p.getDrawInfo()
	if !di.visible {
		return
	}
	di.draw(dc, p)
}

// Hit func.
func (p *Sprite) hit(hc hitContext) (hr hitResult, ok bool) {
	sp, pt := p.getGdiSprite()
	if sp == nil {
		return
	}

	pt = hc.Pos.Sub(pt)
	_, _, _, a := sp.Image().At(pt.X, pt.Y).RGBA()
	if a > 0 {
		return hitResult{Target: p}, true
	}
	return
}

// -------------------------------------------------------------------------------------

func getTrackPos(spr *gdi.Sprite) image.Point {
	pt, _ := grpTrackPos.Get(nil, spr)
	return pt.(image.Point)
}

func doGetTrackPos(ctx objcache.Context, key objcache.Key) (val objcache.Value, err error) {
	spr := key.(*gdi.Sprite)
	pt := spr.GetTrackPos()
	return pt, nil
}

var (
	grpTrackPos *objcache.Group = objcache.NewGroup("tp", 0, doGetTrackPos)
)

// -------------------------------------------------------------------------------------

func (p *sprKey) makeSprite(px, py float64, scale float64, screenW int, screenH int, fs spxfs.Dir) *gdi.Sprite {
	c := p.costume
	img, centerX, centerY := c.needImage(fs)

	scale = scale * p.scale / float64(c.bitmapResolution)
	width, height := img.Size()
	x, y, rotate, box := calcBox(px, py, float64(width), float64(height), centerX, centerY, float64(screenW), float64(screenH), scale, p.direction+c.faceLeft)
	rc := box.Rect()
	dst := ebiten.NewImage(rc.Dx(), rc.Dy())
	DrawBox(dst, img, false, centerX, centerY, x, y, scale, rotate, box)
	spr := gdi.NewSprite(dst, image.Rect(0, 0, rc.Dx(), rc.Dy()))
	spr.Rect = rc
	return spr
}

func DrawBox(target *ebiten.Image, img *ebiten.Image, tran bool, cx, cy, x, y, scale, rotate float64, box *Box) {
	opt := &ebiten.DrawImageOptions{}
	opt.GeoM.Translate(-cx, -cy)
	opt.GeoM.Scale(scale, scale)
	opt.GeoM.Rotate(rotate)
	opt.GeoM.Translate(cx*scale, cy*scale)
	if tran {
		opt.GeoM.Translate(box.X, box.Y)
	}
	target.DrawImage(img, opt)
}

// sprite box
type Box struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

func (b *Box) Rect() image.Rectangle {
	return image.Rect(int(b.X), int(b.Y), int(b.X+b.Width), int(b.Y+b.Height))
}

func calcBox(px, py float64, width, height, centerX, centerY float64, screenW, screenH, scale float64, direction float64) (x float64, y float64, rotate float64, box *Box) {
	if direction == 90 {
		x = screenW/2 + px*scale - centerX*scale
		y = screenH/2 - py*scale - centerY*scale
		box = &Box{x, y, width * scale, height * scale}
	} else {
		x = screenW/2 + px*scale - centerX*scale
		y = screenH/2 - py*scale - centerY*scale
		rotate = toRadian(direction - 90)

		mt := matrix.Identity()
		mt.Translate(-centerX, -centerY)
		mt.Scale(scale, scale)
		mt.Rotate(rotate)
		mt.Translate(x+centerX*scale, y+centerY*scale)

		oldpt := []float64{px, py, px, py + height, px + width, py + height, px + width, py}
		points := mt.TransformPoints(oldpt)
		maxY, minY, maxX, minX := -8000.0, 8000.0, -8000.0, 8000.0
		for i := 0; i < len(points); i += 2 {
			x := points[i]   //+ float64(screenW/2)
			y := points[i+1] //+ float64(screenH/2)

			if maxY < y {
				maxY = y
			}
			if minY > y {
				minY = y
			}
			if maxX < x {
				maxX = x
			}
			if minX > x {
				minX = x
			}
		}
		box = &Box{minX, minY, maxX - minX, maxY - minY}
	}
	return
}
