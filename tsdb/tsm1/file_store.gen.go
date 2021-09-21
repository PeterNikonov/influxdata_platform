// Code generated by file_store.gen.go.tmpl. DO NOT EDIT.

package tsm1

// ReadFloatBlock reads the next block as a set of float values.
func (c *KeyCursor) ReadFloatBlock(buf *[]FloatValue) ([]FloatValue, error) {
LOOP:
	// No matching blocks to decode
	if len(c.current) == 0 {
		return nil, nil
	}

	// First block is the oldest block containing the points we're searching for.
	first := c.current[0]
	*buf = (*buf)[:0]
	var values FloatValues
	values, err := first.r.ReadFloatBlockAt(&first.entry, buf)
	if err != nil {
		return nil, err
	}
	if c.col != nil {
		c.col.GetCounter(floatBlocksDecodedCounter).Add(1)
		c.col.GetCounter(floatBlocksSizeCounter).Add(int64(first.entry.Size))
	}

	// Remove values we already read
	values = values.Exclude(first.readMin, first.readMax)

	// Remove any tombstones
	c.trbuf = first.r.TombstoneRange(c.key, c.trbuf[:0])
	values = excludeTombstonesFloatValues(c.trbuf, values)
	// If there are no values in this first block (all tombstoned or previously read) and
	// we have more potential blocks too search.  Try again.
	if values.Len() == 0 && len(c.current) > 0 {
		c.current = c.current[1:]
		goto LOOP
	}

	// Only one block with this key and time range so return it
	if len(c.current) == 1 {
		if values.Len() > 0 {
			first.markRead(values.MinTime(), values.MaxTime())
		}
		return values, nil
	}

	// Use the current block time range as our overlapping window
	minT, maxT := first.readMin, first.readMax
	if values.Len() > 0 {
		minT, maxT = values.MinTime(), values.MaxTime()
	}
	if c.ascending {
		// Blocks are ordered by generation, we may have values in the past in later blocks, if so,
		// expand the window to include the min time range to ensure values are returned in ascending
		// order
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.MinTime < minT && !cur.read() {
				minT = cur.entry.MinTime
			}
		}

		// Find first block that overlaps our window
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.OverlapsTimeRange(minT, maxT) && !cur.read() {
				// Shrink our window so it's the intersection of the first overlapping block and the
				// first block.  We do this to minimize the region that overlaps and needs to
				// be merged.
				if cur.entry.MaxTime > maxT {
					maxT = cur.entry.MaxTime
				}
				values = values.Include(minT, maxT)
				break
			}
		}

		// Search the remaining blocks that overlap our window and append their values so we can
		// merge them.
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			// Skip this block if it doesn't contain points we looking for or they have already been read
			if !cur.entry.OverlapsTimeRange(minT, maxT) || cur.read() {
				cur.markRead(minT, maxT)
				continue
			}

			var a []FloatValue
			var v FloatValues
			v, err := cur.r.ReadFloatBlockAt(&cur.entry, &a)
			if err != nil {
				return nil, err
			}
			if c.col != nil {
				c.col.GetCounter(floatBlocksDecodedCounter).Add(1)
				c.col.GetCounter(floatBlocksSizeCounter).Add(int64(cur.entry.Size))
			}

			c.trbuf = cur.r.TombstoneRange(c.key, c.trbuf[:0])
			// Remove any tombstoned values
			v = excludeTombstonesFloatValues(c.trbuf, v)

			// Remove values we already read
			v = v.Exclude(cur.readMin, cur.readMax)

			if v.Len() > 0 {
				// Only use values in the overlapping window
				v = v.Include(minT, maxT)
				// Merge the remaining values with the existing
				values = values.Merge(v)
			}
			cur.markRead(minT, maxT)
		}

	} else {
		// Blocks are ordered by generation, we may have values in the past in later blocks, if so,
		// expand the window to include the max time range to ensure values are returned in descending
		// order
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.MaxTime > maxT && !cur.read() {
				maxT = cur.entry.MaxTime
			}
		}

		// Find first block that overlaps our window
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.OverlapsTimeRange(minT, maxT) && !cur.read() {
				// Shrink our window so it's the intersection of the first overlapping block and the
				// first block.  We do this to minimize the region that overlaps and needs to
				// be merged.
				if cur.entry.MinTime < minT {
					minT = cur.entry.MinTime
				}
				values = values.Include(minT, maxT)
				break
			}
		}

		// Search the remaining blocks that overlap our window and append their values so we can
		// merge them.
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			// Skip this block if it doesn't contain points we looking for or they have already been read
			if !cur.entry.OverlapsTimeRange(minT, maxT) || cur.read() {
				cur.markRead(minT, maxT)
				continue
			}

			var a []FloatValue
			var v FloatValues
			v, err := cur.r.ReadFloatBlockAt(&cur.entry, &a)
			if err != nil {
				return nil, err
			}
			if c.col != nil {
				c.col.GetCounter(floatBlocksDecodedCounter).Add(1)
				c.col.GetCounter(floatBlocksSizeCounter).Add(int64(cur.entry.Size))
			}
			c.trbuf = cur.r.TombstoneRange(c.key, c.trbuf[:0])
			// Remove any tombstoned values
			v = excludeTombstonesFloatValues(c.trbuf, v)

			// Remove values we already read
			v = v.Exclude(cur.readMin, cur.readMax)

			// If the block we decoded should have all of it's values included, mark it as read so we
			// don't use it again.
			if v.Len() > 0 {
				v = v.Include(minT, maxT)
				// Merge the remaining values with the existing
				values = v.Merge(values)
			}
			cur.markRead(minT, maxT)
		}
	}

	first.markRead(minT, maxT)

	return values, err
}

func excludeTombstonesFloatValues(t []TimeRange, values FloatValues) FloatValues {
	for i := range t {
		values = values.Exclude(t[i].Min, t[i].Max)
	}
	return values
}

// ReadIntegerBlock reads the next block as a set of integer values.
func (c *KeyCursor) ReadIntegerBlock(buf *[]IntegerValue) ([]IntegerValue, error) {
LOOP:
	// No matching blocks to decode
	if len(c.current) == 0 {
		return nil, nil
	}

	// First block is the oldest block containing the points we're searching for.
	first := c.current[0]
	*buf = (*buf)[:0]
	var values IntegerValues
	values, err := first.r.ReadIntegerBlockAt(&first.entry, buf)
	if err != nil {
		return nil, err
	}
	if c.col != nil {
		c.col.GetCounter(integerBlocksDecodedCounter).Add(1)
		c.col.GetCounter(integerBlocksSizeCounter).Add(int64(first.entry.Size))
	}

	// Remove values we already read
	values = values.Exclude(first.readMin, first.readMax)

	// Remove any tombstones
	c.trbuf = first.r.TombstoneRange(c.key, c.trbuf[:0])
	values = excludeTombstonesIntegerValues(c.trbuf, values)
	// If there are no values in this first block (all tombstoned or previously read) and
	// we have more potential blocks too search.  Try again.
	if values.Len() == 0 && len(c.current) > 0 {
		c.current = c.current[1:]
		goto LOOP
	}

	// Only one block with this key and time range so return it
	if len(c.current) == 1 {
		if values.Len() > 0 {
			first.markRead(values.MinTime(), values.MaxTime())
		}
		return values, nil
	}

	// Use the current block time range as our overlapping window
	minT, maxT := first.readMin, first.readMax
	if values.Len() > 0 {
		minT, maxT = values.MinTime(), values.MaxTime()
	}
	if c.ascending {
		// Blocks are ordered by generation, we may have values in the past in later blocks, if so,
		// expand the window to include the min time range to ensure values are returned in ascending
		// order
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.MinTime < minT && !cur.read() {
				minT = cur.entry.MinTime
			}
		}

		// Find first block that overlaps our window
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.OverlapsTimeRange(minT, maxT) && !cur.read() {
				// Shrink our window so it's the intersection of the first overlapping block and the
				// first block.  We do this to minimize the region that overlaps and needs to
				// be merged.
				if cur.entry.MaxTime > maxT {
					maxT = cur.entry.MaxTime
				}
				values = values.Include(minT, maxT)
				break
			}
		}

		// Search the remaining blocks that overlap our window and append their values so we can
		// merge them.
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			// Skip this block if it doesn't contain points we looking for or they have already been read
			if !cur.entry.OverlapsTimeRange(minT, maxT) || cur.read() {
				cur.markRead(minT, maxT)
				continue
			}

			var a []IntegerValue
			var v IntegerValues
			v, err := cur.r.ReadIntegerBlockAt(&cur.entry, &a)
			if err != nil {
				return nil, err
			}
			if c.col != nil {
				c.col.GetCounter(integerBlocksDecodedCounter).Add(1)
				c.col.GetCounter(integerBlocksSizeCounter).Add(int64(cur.entry.Size))
			}

			c.trbuf = cur.r.TombstoneRange(c.key, c.trbuf[:0])
			// Remove any tombstoned values
			v = excludeTombstonesIntegerValues(c.trbuf, v)

			// Remove values we already read
			v = v.Exclude(cur.readMin, cur.readMax)

			if v.Len() > 0 {
				// Only use values in the overlapping window
				v = v.Include(minT, maxT)
				// Merge the remaining values with the existing
				values = values.Merge(v)
			}
			cur.markRead(minT, maxT)
		}

	} else {
		// Blocks are ordered by generation, we may have values in the past in later blocks, if so,
		// expand the window to include the max time range to ensure values are returned in descending
		// order
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.MaxTime > maxT && !cur.read() {
				maxT = cur.entry.MaxTime
			}
		}

		// Find first block that overlaps our window
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.OverlapsTimeRange(minT, maxT) && !cur.read() {
				// Shrink our window so it's the intersection of the first overlapping block and the
				// first block.  We do this to minimize the region that overlaps and needs to
				// be merged.
				if cur.entry.MinTime < minT {
					minT = cur.entry.MinTime
				}
				values = values.Include(minT, maxT)
				break
			}
		}

		// Search the remaining blocks that overlap our window and append their values so we can
		// merge them.
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			// Skip this block if it doesn't contain points we looking for or they have already been read
			if !cur.entry.OverlapsTimeRange(minT, maxT) || cur.read() {
				cur.markRead(minT, maxT)
				continue
			}

			var a []IntegerValue
			var v IntegerValues
			v, err := cur.r.ReadIntegerBlockAt(&cur.entry, &a)
			if err != nil {
				return nil, err
			}
			if c.col != nil {
				c.col.GetCounter(integerBlocksDecodedCounter).Add(1)
				c.col.GetCounter(integerBlocksSizeCounter).Add(int64(cur.entry.Size))
			}
			c.trbuf = cur.r.TombstoneRange(c.key, c.trbuf[:0])
			// Remove any tombstoned values
			v = excludeTombstonesIntegerValues(c.trbuf, v)

			// Remove values we already read
			v = v.Exclude(cur.readMin, cur.readMax)

			// If the block we decoded should have all of it's values included, mark it as read so we
			// don't use it again.
			if v.Len() > 0 {
				v = v.Include(minT, maxT)
				// Merge the remaining values with the existing
				values = v.Merge(values)
			}
			cur.markRead(minT, maxT)
		}
	}

	first.markRead(minT, maxT)

	return values, err
}

func excludeTombstonesIntegerValues(t []TimeRange, values IntegerValues) IntegerValues {
	for i := range t {
		values = values.Exclude(t[i].Min, t[i].Max)
	}
	return values
}

// ReadUnsignedBlock reads the next block as a set of unsigned values.
func (c *KeyCursor) ReadUnsignedBlock(buf *[]UnsignedValue) ([]UnsignedValue, error) {
LOOP:
	// No matching blocks to decode
	if len(c.current) == 0 {
		return nil, nil
	}

	// First block is the oldest block containing the points we're searching for.
	first := c.current[0]
	*buf = (*buf)[:0]
	var values UnsignedValues
	values, err := first.r.ReadUnsignedBlockAt(&first.entry, buf)
	if err != nil {
		return nil, err
	}
	if c.col != nil {
		c.col.GetCounter(unsignedBlocksDecodedCounter).Add(1)
		c.col.GetCounter(unsignedBlocksSizeCounter).Add(int64(first.entry.Size))
	}

	// Remove values we already read
	values = values.Exclude(first.readMin, first.readMax)

	// Remove any tombstones
	c.trbuf = first.r.TombstoneRange(c.key, c.trbuf[:0])
	values = excludeTombstonesUnsignedValues(c.trbuf, values)
	// If there are no values in this first block (all tombstoned or previously read) and
	// we have more potential blocks too search.  Try again.
	if values.Len() == 0 && len(c.current) > 0 {
		c.current = c.current[1:]
		goto LOOP
	}

	// Only one block with this key and time range so return it
	if len(c.current) == 1 {
		if values.Len() > 0 {
			first.markRead(values.MinTime(), values.MaxTime())
		}
		return values, nil
	}

	// Use the current block time range as our overlapping window
	minT, maxT := first.readMin, first.readMax
	if values.Len() > 0 {
		minT, maxT = values.MinTime(), values.MaxTime()
	}
	if c.ascending {
		// Blocks are ordered by generation, we may have values in the past in later blocks, if so,
		// expand the window to include the min time range to ensure values are returned in ascending
		// order
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.MinTime < minT && !cur.read() {
				minT = cur.entry.MinTime
			}
		}

		// Find first block that overlaps our window
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.OverlapsTimeRange(minT, maxT) && !cur.read() {
				// Shrink our window so it's the intersection of the first overlapping block and the
				// first block.  We do this to minimize the region that overlaps and needs to
				// be merged.
				if cur.entry.MaxTime > maxT {
					maxT = cur.entry.MaxTime
				}
				values = values.Include(minT, maxT)
				break
			}
		}

		// Search the remaining blocks that overlap our window and append their values so we can
		// merge them.
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			// Skip this block if it doesn't contain points we looking for or they have already been read
			if !cur.entry.OverlapsTimeRange(minT, maxT) || cur.read() {
				cur.markRead(minT, maxT)
				continue
			}

			var a []UnsignedValue
			var v UnsignedValues
			v, err := cur.r.ReadUnsignedBlockAt(&cur.entry, &a)
			if err != nil {
				return nil, err
			}
			if c.col != nil {
				c.col.GetCounter(unsignedBlocksDecodedCounter).Add(1)
				c.col.GetCounter(unsignedBlocksSizeCounter).Add(int64(cur.entry.Size))
			}

			c.trbuf = cur.r.TombstoneRange(c.key, c.trbuf[:0])
			// Remove any tombstoned values
			v = excludeTombstonesUnsignedValues(c.trbuf, v)

			// Remove values we already read
			v = v.Exclude(cur.readMin, cur.readMax)

			if v.Len() > 0 {
				// Only use values in the overlapping window
				v = v.Include(minT, maxT)
				// Merge the remaining values with the existing
				values = values.Merge(v)
			}
			cur.markRead(minT, maxT)
		}

	} else {
		// Blocks are ordered by generation, we may have values in the past in later blocks, if so,
		// expand the window to include the max time range to ensure values are returned in descending
		// order
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.MaxTime > maxT && !cur.read() {
				maxT = cur.entry.MaxTime
			}
		}

		// Find first block that overlaps our window
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.OverlapsTimeRange(minT, maxT) && !cur.read() {
				// Shrink our window so it's the intersection of the first overlapping block and the
				// first block.  We do this to minimize the region that overlaps and needs to
				// be merged.
				if cur.entry.MinTime < minT {
					minT = cur.entry.MinTime
				}
				values = values.Include(minT, maxT)
				break
			}
		}

		// Search the remaining blocks that overlap our window and append their values so we can
		// merge them.
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			// Skip this block if it doesn't contain points we looking for or they have already been read
			if !cur.entry.OverlapsTimeRange(minT, maxT) || cur.read() {
				cur.markRead(minT, maxT)
				continue
			}

			var a []UnsignedValue
			var v UnsignedValues
			v, err := cur.r.ReadUnsignedBlockAt(&cur.entry, &a)
			if err != nil {
				return nil, err
			}
			if c.col != nil {
				c.col.GetCounter(unsignedBlocksDecodedCounter).Add(1)
				c.col.GetCounter(unsignedBlocksSizeCounter).Add(int64(cur.entry.Size))
			}
			c.trbuf = cur.r.TombstoneRange(c.key, c.trbuf[:0])
			// Remove any tombstoned values
			v = excludeTombstonesUnsignedValues(c.trbuf, v)

			// Remove values we already read
			v = v.Exclude(cur.readMin, cur.readMax)

			// If the block we decoded should have all of it's values included, mark it as read so we
			// don't use it again.
			if v.Len() > 0 {
				v = v.Include(minT, maxT)
				// Merge the remaining values with the existing
				values = v.Merge(values)
			}
			cur.markRead(minT, maxT)
		}
	}

	first.markRead(minT, maxT)

	return values, err
}

func excludeTombstonesUnsignedValues(t []TimeRange, values UnsignedValues) UnsignedValues {
	for i := range t {
		values = values.Exclude(t[i].Min, t[i].Max)
	}
	return values
}

// ReadStringBlock reads the next block as a set of string values.
func (c *KeyCursor) ReadStringBlock(buf *[]StringValue) ([]StringValue, error) {
LOOP:
	// No matching blocks to decode
	if len(c.current) == 0 {
		return nil, nil
	}

	// First block is the oldest block containing the points we're searching for.
	first := c.current[0]
	*buf = (*buf)[:0]
	var values StringValues
	values, err := first.r.ReadStringBlockAt(&first.entry, buf)
	if err != nil {
		return nil, err
	}
	if c.col != nil {
		c.col.GetCounter(stringBlocksDecodedCounter).Add(1)
		c.col.GetCounter(stringBlocksSizeCounter).Add(int64(first.entry.Size))
	}

	// Remove values we already read
	values = values.Exclude(first.readMin, first.readMax)

	// Remove any tombstones
	c.trbuf = first.r.TombstoneRange(c.key, c.trbuf[:0])
	values = excludeTombstonesStringValues(c.trbuf, values)
	// If there are no values in this first block (all tombstoned or previously read) and
	// we have more potential blocks too search.  Try again.
	if values.Len() == 0 && len(c.current) > 0 {
		c.current = c.current[1:]
		goto LOOP
	}

	// Only one block with this key and time range so return it
	if len(c.current) == 1 {
		if values.Len() > 0 {
			first.markRead(values.MinTime(), values.MaxTime())
		}
		return values, nil
	}

	// Use the current block time range as our overlapping window
	minT, maxT := first.readMin, first.readMax
	if values.Len() > 0 {
		minT, maxT = values.MinTime(), values.MaxTime()
	}
	if c.ascending {
		// Blocks are ordered by generation, we may have values in the past in later blocks, if so,
		// expand the window to include the min time range to ensure values are returned in ascending
		// order
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.MinTime < minT && !cur.read() {
				minT = cur.entry.MinTime
			}
		}

		// Find first block that overlaps our window
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.OverlapsTimeRange(minT, maxT) && !cur.read() {
				// Shrink our window so it's the intersection of the first overlapping block and the
				// first block.  We do this to minimize the region that overlaps and needs to
				// be merged.
				if cur.entry.MaxTime > maxT {
					maxT = cur.entry.MaxTime
				}
				values = values.Include(minT, maxT)
				break
			}
		}

		// Search the remaining blocks that overlap our window and append their values so we can
		// merge them.
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			// Skip this block if it doesn't contain points we looking for or they have already been read
			if !cur.entry.OverlapsTimeRange(minT, maxT) || cur.read() {
				cur.markRead(minT, maxT)
				continue
			}

			var a []StringValue
			var v StringValues
			v, err := cur.r.ReadStringBlockAt(&cur.entry, &a)
			if err != nil {
				return nil, err
			}
			if c.col != nil {
				c.col.GetCounter(stringBlocksDecodedCounter).Add(1)
				c.col.GetCounter(stringBlocksSizeCounter).Add(int64(cur.entry.Size))
			}

			c.trbuf = cur.r.TombstoneRange(c.key, c.trbuf[:0])
			// Remove any tombstoned values
			v = excludeTombstonesStringValues(c.trbuf, v)

			// Remove values we already read
			v = v.Exclude(cur.readMin, cur.readMax)

			if v.Len() > 0 {
				// Only use values in the overlapping window
				v = v.Include(minT, maxT)
				// Merge the remaining values with the existing
				values = values.Merge(v)
			}
			cur.markRead(minT, maxT)
		}

	} else {
		// Blocks are ordered by generation, we may have values in the past in later blocks, if so,
		// expand the window to include the max time range to ensure values are returned in descending
		// order
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.MaxTime > maxT && !cur.read() {
				maxT = cur.entry.MaxTime
			}
		}

		// Find first block that overlaps our window
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.OverlapsTimeRange(minT, maxT) && !cur.read() {
				// Shrink our window so it's the intersection of the first overlapping block and the
				// first block.  We do this to minimize the region that overlaps and needs to
				// be merged.
				if cur.entry.MinTime < minT {
					minT = cur.entry.MinTime
				}
				values = values.Include(minT, maxT)
				break
			}
		}

		// Search the remaining blocks that overlap our window and append their values so we can
		// merge them.
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			// Skip this block if it doesn't contain points we looking for or they have already been read
			if !cur.entry.OverlapsTimeRange(minT, maxT) || cur.read() {
				cur.markRead(minT, maxT)
				continue
			}

			var a []StringValue
			var v StringValues
			v, err := cur.r.ReadStringBlockAt(&cur.entry, &a)
			if err != nil {
				return nil, err
			}
			if c.col != nil {
				c.col.GetCounter(stringBlocksDecodedCounter).Add(1)
				c.col.GetCounter(stringBlocksSizeCounter).Add(int64(cur.entry.Size))
			}
			c.trbuf = cur.r.TombstoneRange(c.key, c.trbuf[:0])
			// Remove any tombstoned values
			v = excludeTombstonesStringValues(c.trbuf, v)

			// Remove values we already read
			v = v.Exclude(cur.readMin, cur.readMax)

			// If the block we decoded should have all of it's values included, mark it as read so we
			// don't use it again.
			if v.Len() > 0 {
				v = v.Include(minT, maxT)
				// Merge the remaining values with the existing
				values = v.Merge(values)
			}
			cur.markRead(minT, maxT)
		}
	}

	first.markRead(minT, maxT)

	return values, err
}

func excludeTombstonesStringValues(t []TimeRange, values StringValues) StringValues {
	for i := range t {
		values = values.Exclude(t[i].Min, t[i].Max)
	}
	return values
}

// ReadBooleanBlock reads the next block as a set of boolean values.
func (c *KeyCursor) ReadBooleanBlock(buf *[]BooleanValue) ([]BooleanValue, error) {
LOOP:
	// No matching blocks to decode
	if len(c.current) == 0 {
		return nil, nil
	}

	// First block is the oldest block containing the points we're searching for.
	first := c.current[0]
	*buf = (*buf)[:0]
	var values BooleanValues
	values, err := first.r.ReadBooleanBlockAt(&first.entry, buf)
	if err != nil {
		return nil, err
	}
	if c.col != nil {
		c.col.GetCounter(booleanBlocksDecodedCounter).Add(1)
		c.col.GetCounter(booleanBlocksSizeCounter).Add(int64(first.entry.Size))
	}

	// Remove values we already read
	values = values.Exclude(first.readMin, first.readMax)

	// Remove any tombstones
	c.trbuf = first.r.TombstoneRange(c.key, c.trbuf[:0])
	values = excludeTombstonesBooleanValues(c.trbuf, values)
	// If there are no values in this first block (all tombstoned or previously read) and
	// we have more potential blocks too search.  Try again.
	if values.Len() == 0 && len(c.current) > 0 {
		c.current = c.current[1:]
		goto LOOP
	}

	// Only one block with this key and time range so return it
	if len(c.current) == 1 {
		if values.Len() > 0 {
			first.markRead(values.MinTime(), values.MaxTime())
		}
		return values, nil
	}

	// Use the current block time range as our overlapping window
	minT, maxT := first.readMin, first.readMax
	if values.Len() > 0 {
		minT, maxT = values.MinTime(), values.MaxTime()
	}
	if c.ascending {
		// Blocks are ordered by generation, we may have values in the past in later blocks, if so,
		// expand the window to include the min time range to ensure values are returned in ascending
		// order
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.MinTime < minT && !cur.read() {
				minT = cur.entry.MinTime
			}
		}

		// Find first block that overlaps our window
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.OverlapsTimeRange(minT, maxT) && !cur.read() {
				// Shrink our window so it's the intersection of the first overlapping block and the
				// first block.  We do this to minimize the region that overlaps and needs to
				// be merged.
				if cur.entry.MaxTime > maxT {
					maxT = cur.entry.MaxTime
				}
				values = values.Include(minT, maxT)
				break
			}
		}

		// Search the remaining blocks that overlap our window and append their values so we can
		// merge them.
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			// Skip this block if it doesn't contain points we looking for or they have already been read
			if !cur.entry.OverlapsTimeRange(minT, maxT) || cur.read() {
				cur.markRead(minT, maxT)
				continue
			}

			var a []BooleanValue
			var v BooleanValues
			v, err := cur.r.ReadBooleanBlockAt(&cur.entry, &a)
			if err != nil {
				return nil, err
			}
			if c.col != nil {
				c.col.GetCounter(booleanBlocksDecodedCounter).Add(1)
				c.col.GetCounter(booleanBlocksSizeCounter).Add(int64(cur.entry.Size))
			}

			c.trbuf = cur.r.TombstoneRange(c.key, c.trbuf[:0])
			// Remove any tombstoned values
			v = excludeTombstonesBooleanValues(c.trbuf, v)

			// Remove values we already read
			v = v.Exclude(cur.readMin, cur.readMax)

			if v.Len() > 0 {
				// Only use values in the overlapping window
				v = v.Include(minT, maxT)
				// Merge the remaining values with the existing
				values = values.Merge(v)
			}
			cur.markRead(minT, maxT)
		}

	} else {
		// Blocks are ordered by generation, we may have values in the past in later blocks, if so,
		// expand the window to include the max time range to ensure values are returned in descending
		// order
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.MaxTime > maxT && !cur.read() {
				maxT = cur.entry.MaxTime
			}
		}

		// Find first block that overlaps our window
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			if cur.entry.OverlapsTimeRange(minT, maxT) && !cur.read() {
				// Shrink our window so it's the intersection of the first overlapping block and the
				// first block.  We do this to minimize the region that overlaps and needs to
				// be merged.
				if cur.entry.MinTime < minT {
					minT = cur.entry.MinTime
				}
				values = values.Include(minT, maxT)
				break
			}
		}

		// Search the remaining blocks that overlap our window and append their values so we can
		// merge them.
		for i := 1; i < len(c.current); i++ {
			cur := c.current[i]
			// Skip this block if it doesn't contain points we looking for or they have already been read
			if !cur.entry.OverlapsTimeRange(minT, maxT) || cur.read() {
				cur.markRead(minT, maxT)
				continue
			}

			var a []BooleanValue
			var v BooleanValues
			v, err := cur.r.ReadBooleanBlockAt(&cur.entry, &a)
			if err != nil {
				return nil, err
			}
			if c.col != nil {
				c.col.GetCounter(booleanBlocksDecodedCounter).Add(1)
				c.col.GetCounter(booleanBlocksSizeCounter).Add(int64(cur.entry.Size))
			}
			c.trbuf = cur.r.TombstoneRange(c.key, c.trbuf[:0])
			// Remove any tombstoned values
			v = excludeTombstonesBooleanValues(c.trbuf, v)

			// Remove values we already read
			v = v.Exclude(cur.readMin, cur.readMax)

			// If the block we decoded should have all of it's values included, mark it as read so we
			// don't use it again.
			if v.Len() > 0 {
				v = v.Include(minT, maxT)
				// Merge the remaining values with the existing
				values = v.Merge(values)
			}
			cur.markRead(minT, maxT)
		}
	}

	first.markRead(minT, maxT)

	return values, err
}

func excludeTombstonesBooleanValues(t []TimeRange, values BooleanValues) BooleanValues {
	for i := range t {
		values = values.Exclude(t[i].Min, t[i].Max)
	}
	return values
}
