package microui

// poolInit finds the pool slot with the lowest LastUpdate frame (least
// recently used), assigns id to it, marks it as updated this frame, and
// returns its index. Panics if no slot is available (should not happen —
// all slots start at LastUpdate=0 and Frame increments monotonically).
func (ctx *Context) poolInit(items []PoolItem, id ID) int {
	n := -1
	f := ctx.Frame
	for i := range items {
		if items[i].LastUpdate < f {
			f = items[i].LastUpdate
			n = i
		}
	}
	expect(n > -1, "poolInit: no free slot")
	items[n].ID = id
	ctx.poolUpdate(items, n)
	return n
}

// poolGet returns the index of the pool slot whose ID matches id, or -1.
func (ctx *Context) poolGet(items []PoolItem, id ID) int {
	for i := range items {
		if items[i].ID == id {
			return i
		}
	}
	return -1
}

// poolUpdate marks slot idx as updated this frame.
func (ctx *Context) poolUpdate(items []PoolItem, idx int) {
	items[idx].LastUpdate = ctx.Frame
}
