package common

func ChunkArray(arr []map[string]interface{}, size int) [][]map[string]interface{} {
	var chunks [][]map[string]interface{}
	for size < len(arr) {
		arr, chunks = arr[size:], append(chunks, arr[0:size])
	}
	chunks = append(chunks, arr)
	return chunks
}