package main

import "github.com/tanema/gween"

type Action struct {
	nexts    []func(g *Game)
	onChange func(float32)
	onFinish []func()
}

func (a *Action) addOnFinish(f func()) {
	if a.onFinish == nil {
		a.onFinish = make([]func(), 0)
	}
	a.onFinish = append(a.onFinish, f)
}

func (a *Action) next(t *gween.Tween) *Action {
	action := Action{}
	if a.nexts == nil {
		a.nexts = make([]func(g *Game), 0)
	}
	a.nexts = append(a.nexts,
		func(g *Game) {
			g.Tweens[t] = action
		})
	return &action
}
