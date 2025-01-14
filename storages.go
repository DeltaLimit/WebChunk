/*
	WebChunk, web server for block game maps
	Copyright (C) 2022 Maxim Zhuchkov

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU Affero General Public License as published
	by the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU Affero General Public License for more details.

	You should have received a copy of the GNU Affero General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.

	Contact me via mail: q3.max.2011@yandex.ru or Discord: MaX#6717
*/

package main

import (
	"context"
	"errors"
	"log"
	"strings"

	"github.com/Tnze/go-mc/level"
	"github.com/Tnze/go-mc/nbt"
	"github.com/Tnze/go-mc/save"
	"github.com/maxsupermanhd/WebChunk/chunkStorage"
	"github.com/maxsupermanhd/WebChunk/chunkStorage/postgresChunkStorage"
	"github.com/maxsupermanhd/WebChunk/proxy"
)

var (
	errStorageTypeNotImplemented = errors.New("storage type not implemented")
)

func initStorage(t, a string) (driver chunkStorage.ChunkStorage, err error) {
	switch t {
	case "postgres":
		driver, err = postgresChunkStorage.NewPostgresChunkStorage(context.Background(), a)
		if err != nil {
			return nil, err
		}
		return driver, nil
	default:
		return nil, errStorageTypeNotImplemented
	}
}

func findCapableStorage(arr []chunkStorage.Storage, pref string) chunkStorage.ChunkStorage {
	var s chunkStorage.ChunkStorage
	var sf chunkStorage.ChunkStorage
	for i := range arr {
		if arr[i].Driver == nil || arr[i].Name == "" {
			continue
		}
		if arr[i].Name == pref {
			s = arr[i].Driver
			break
		}
		a := arr[i].Driver.GetAbilities()
		if a.CanCreateWorldsDimensions &&
			a.CanAddChunks &&
			a.CanPreserveOldChunks {
			sf = arr[i].Driver
		}
	}
	if s == nil {
		s = sf
	}
	return s
}

func chunkConsumer(c chan *proxy.ProxiedChunk) {
	for r := range c {
		route, ok := loadedConfig.Routes[r.Username]
		if !ok {
			log.Printf("Got UNKNOWN chunk [%v] from [%v] by [%v]", r.Pos, r.Server, r.Username)
		}
		log.Printf("Got chunk [%v] from [%v] by [%v] (%d sections) (%d block entities)", r.Pos, r.Server, r.Username, len(r.Data.Sections), len(r.Data.BlockEntity))
		if route.World == "" {
			route.World = r.Server
		}
		if route.Dimension == "" {
			route.Dimension = r.Dimension
		}
		w, s, err := chunkStorage.GetWorldStorage(storages, route.World)
		if err != nil {
			log.Println("Failed to lookup world storage: ", err)
			break
		}
		var d *chunkStorage.DimStruct
		if w == nil || s == nil {
			s = findCapableStorage(storages, route.Storage)
			if s == nil {
				log.Printf("Failed to find storage that has world [%s], named [%s] or has ability to add chunks, chunk [%v] from [%v] by [%v] is LOST.", route.World, route.Storage, r.Pos, r.Server, r.Username)
				continue
			}
			w, err = s.AddWorld(route.World, r.Server)
			if err != nil {
				log.Printf("Failed to add world: %s", err.Error())
				continue
			}
		}
		d, err = s.GetDimension(w.Name, route.Dimension)
		if err != nil {
			log.Printf("Failed to get dim: %s", err.Error())
			continue
		}
		if d == nil {
			d = &chunkStorage.DimStruct{
				Name:       route.Dimension,
				Alias:      strings.TrimPrefix(route.Dimension, "minecraft:"),
				World:      w.Name,
				Spawnpoint: [3]int64{0, 64, 0},
				LowestY:    int(r.DimensionLowestY),
				BuildLimit: int(r.DimensionBuildLimit),
			}
			d, err = s.AddDimension(*d)
			if err != nil {
				log.Printf("Failed to add dim: %s", err.Error())
				continue
			}
		}
		if d == nil {
			log.Println("d is nill")
			continue
		}
		if w == nil {
			log.Println("w is nill")
			continue
		}
		if d.World != w.Name {
			log.Printf("SUS dim's wname != world's name [%s] [%s]", d.World, w.Name)
			continue
		}
		nbtEmptyList := nbt.RawMessage{
			Type: nbt.TagList,
			Data: []byte{0, 0, 0, 0, 0},
		}
		var data save.Chunk
		data.XPos = int32(r.Pos.X)
		data.ZPos = int32(r.Pos.Z)
		level.ChunkToSave(&r.Data, &data)
		// for iiii, cccc := range data.Sections {
		// 	log.Printf("Section %d palette len %d indexes len %d", iiii, len(cccc.BlockStates.Palette), len(cccc.BlockStates.Data))
		// }
		data.BlockEntities = nbtEmptyList
		data.Structures = nbtEmptyList
		data.Heightmaps = struct {
			MotionBlocking         []uint64 "nbt:\"MOTION_BLOCKING\""
			MotionBlockingNoLeaves []uint64 "nbt:\"MOTION_BLOCKING_NO_LEAVES\""
			OceanFloor             []uint64 "nbt:\"OCEAN_FLOOR\""
			WorldSurface           []uint64 "nbt:\"WORLD_SURFACE\""
		}{
			MotionBlocking:         r.Data.HeightMaps.MotionBlocking.Raw(),
			MotionBlockingNoLeaves: r.Data.HeightMaps.MotionBlockingNoLeaves.Raw(),
			OceanFloor:             r.Data.HeightMaps.OceanFloor.Raw(),
			WorldSurface:           r.Data.HeightMaps.WorldSurface.Raw(),
		}
		data.BlockTicks = nbtEmptyList
		data.FluidTicks = nbtEmptyList
		data.PostProcessing = nbtEmptyList
		err = s.AddChunk(w.Name, d.Name, int64(r.Pos.X), int64(r.Pos.Z), data)
		if err != nil {
			log.Printf("Failed to save chunk: %s", err.Error())
		}
	}
}

// func chunkLevelToSave(in *level.Chunk, lowestY int32, cx, cz int32) (*save.Chunk, error) {
// 	spew.Dump(in)
// 	out := save.Chunk{
// 		DataVersion:   2865, // was at the moment of writing
// 		XPos:          int32(cx),
// 		YPos:          lowestY / 16,
// 		ZPos:          int32(cz),
// 		BlockEntities: nbt.RawMessage{},
// 		Structures:    nbt.RawMessage{}, // we will never get those
// 		Heightmaps: struct {
// 			MotionBlocking         []int64 "nbt:\"MOTION_BLOCKING\""
// 			MotionBlockingNoLeaves []int64 "nbt:\"MOTION_BLOCKING_NO_LEAVES\""
// 			OceanFloor             []int64 "nbt:\"OCEAN_FLOOR\""
// 			WorldSurface           []int64 "nbt:\"WORLD_SURFACE\""
// 		}{
// 			MotionBlocking:         []int64{}, //*(*[]int64)(unsafe.Pointer(in.HeightMaps.MotionBlocking)),
// 			MotionBlockingNoLeaves: []int64{},
// 			OceanFloor:             []int64{},
// 			WorldSurface:           []int64{}, //*(*[]int64)(unsafe.Pointer(in.HeightMaps.WorldSurface)),
// 		},
// 		Sections: []save.Section{},
// 	}
// 	for y, s := range in.Sections {
// 		o := save.Section{
// 			Y: int8(y + int(out.YPos)),
// 			BlockStates: struct {
// 				Palette []save.BlockState "nbt:\"palette\""
// 				Data    []int64           "nbt:\"data\""
// 			}{
// 				Palette: []save.BlockState{},
// 				Data:    []int64{},
// 			},
// 			Biomes: struct {
// 				Palette []string "nbt:\"palette\""
// 				Data    []int64  "nbt:\"data\""
// 			}{
// 				Palette: []string{},
// 				Data:    []int64{},
// 			},
// 			SkyLight:   []byte{0},
// 			BlockLight: []byte{0},
// 		}

// 		convbuf := bytes.NewBuffer([]byte{})
// 		s.States.WriteTo(convbuf)

// 		// blockstates
// 		// if s.BlockCount == 0 {
// 		// 	o.BlockStates.Palette = append(o.BlockStates.Palette, save.BlockState{
// 		// 		Name: "minecraft:air",
// 		// 		Properties: nbt.RawMessage{
// 		// 			Type: nbt.TagCompound,
// 		// 			Data: []byte{nbt.TagEnd},
// 		// 		},
// 		// 	})
// 		// } else {
// 		// 	statesPalette := []int{}
// 		// 	statesIndexes := [4096]int64{}
// 		// 	for i := 0; i < 16*16*16; i++ {
// 		// 		addState := s.States.Get(i)
// 		// 		foundstate := -1
// 		// 		for ii := range statesPalette {
// 		// 			if statesPalette[ii] == addState {
// 		// 				foundstate = ii
// 		// 				break
// 		// 			}
// 		// 		}
// 		// 		if foundstate == -1 {
// 		// 			statesPalette = append(statesPalette, addState)
// 		// 			foundstate = len(statesPalette) - 1
// 		// 		}
// 		// 		statesIndexes[i] = int64(foundstate)
// 		// 	}
// 		// 	for i := range statesPalette {
// 		// 		b := block.StateList[statesPalette[i]]
// 		// 		addPalette := save.BlockState{
// 		// 			Name: b.ID(),
// 		// 			Properties: nbt.RawMessage{
// 		// 				Type: nbt.TagCompound,
// 		// 				Data: []byte{nbt.TagEnd},
// 		// 			},
// 		// 		}
// 		// 		dat, err := nbt.Marshal(b)
// 		// 		if err != nil {
// 		// 			return nil, err
// 		// 		}
// 		// 		if len(dat) == 4 {
// 		// 			dat = []byte{0x0a, 0x00}
// 		// 		}
// 		// 		addPalette.Properties.Data = dat
// 		// 		o.BlockStates.Palette = append(o.BlockStates.Palette, addPalette)
// 		// 	}
// 		// 	if len(o.BlockStates.Palette) > 1 {
// 		// 		sizeBits := int64(0)
// 		// 		for i := int64(0); i < 32; i++ {
// 		// 			if len(statesPalette)&(1<<i) != 0 {
// 		// 				sizeBits = i + 1
// 		// 			}
// 		// 		}
// 		// 		if sizeBits < 4 {
// 		// 			sizeBits = 4
// 		// 		}
// 		// 		haveBits := int64(64)
// 		// 		currData := int64(0)
// 		// 		for i := range statesIndexes {
// 		// 			if haveBits < sizeBits {
// 		// 				if haveBits == 0 {
// 		// 					o.BlockStates.Data = append(o.BlockStates.Data, int64(currData))
// 		// 					haveBits = 64
// 		// 					currData = 0
// 		// 				} else {
// 		// 					leftBits := sizeBits - haveBits
// 		// 					currData = currData | (statesIndexes[i] >> leftBits)
// 		// 					o.BlockStates.Data = append(o.BlockStates.Data, int64(currData))
// 		// 					haveBits = 64 + haveBits
// 		// 					currData = 0
// 		// 				}
// 		// 			}
// 		// 			currData = currData | (statesIndexes[i])<<(haveBits-sizeBits)
// 		// 			haveBits -= sizeBits
// 		// 		}
// 		// 		if haveBits != 64 {
// 		// 			o.BlockStates.Data = append(o.BlockStates.Data, currData)
// 		// 		}
// 		// 	}
// 		// }
// 		// biomes
// 		// heightmaps
// 		out.Sections = append(out.Sections, o)
// 	}
// 	log.Printf("Saved %d sections", len(out.Sections))
// 	spew.Dump(out)
// 	return &out, nil
// }
