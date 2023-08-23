package database

var (
	DKMaps = map[int16][]int16{
		18: {18, 193, 200}, 19: {19, 194, 201}, 25: {25, 195, 202}, 26: {26, 196, 203}, 27: {27, 197, 204}, 29: {29, 198, 205}, 30: {30, 199, 206}, // Normal Maps
		193: {18, 193, 200}, 194: {19, 194, 201}, 195: {25, 195, 202}, 196: {26, 196, 203}, 197: {27, 197, 204}, 198: {29, 198, 205}, 199: {30, 199, 206}, // DK Maps
		200: {18, 193, 200}, 201: {19, 194, 201}, 202: {25, 195, 202}, 203: {26, 196, 203}, 204: {27, 197, 204}, 205: {29, 198, 205}, 206: {30, 199, 206}, // Normal Maps
	}

	sharedMaps = []int16{1, 254}

	DungeonZones = []int16{229}

	DarkZones = []int16{160, 161, 162, 163, 164, 165, 166, 167, 168, 169, 170, 233}

	PvPZones     = []int16{12, 17, 28, 108, 109, 110, 111, 112, 230}
	unlockedMaps = []int16{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 60, 70, 71, 72, 73, 74, 75, 89, 90, 91, 93, 94, 95, 96, 97, 98, 99, 100, 101, 102, 105, 106, 107, 108, 109, 110, 111, 112, 120, 121, 122, 123, 150, 151, 152, 160, 161, 162, 163, 164, 165, 166, 167, 168, 169, 170, 193, 194, 195, 196, 197, 198, 199, 200, 201, 202, 203, 204, 205, 206, 212, 213, 214, 221, 222, 223, 224, 225, 226, 227, 228, 229, 230, 231, 232, 233, 235, 236, 237, 238, 239, 240, 241, 242, 243, 244, 246, 247, 248, 249, 252, 253, 254, 255}

	PVPServers     = []int16{6, 7}
	LoseEXPServers = []int16{5}
)