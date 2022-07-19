package utils

import (
	"os"
	"strconv"
	"strings"
)

func Exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

/*
Translate the following C# function to go
public static void NumericalSort(string[] ar)
        {
            Regex rgx = new Regex("([^0-9]*)([0-9]+)");
            Array.Sort(ar, (a, b) =>
            {
                var ma = rgx.Matches(a);
                var mb = rgx.Matches(b);
                for (int i = 0; i < ma.Count; ++i)
                {
                    int ret = ma[i].Groups[1].Value.CompareTo(mb[i].Groups[1].Value);
                    if (ret != 0)
                        return ret;

                    ret = int.Parse(ma[i].Groups[2].Value) - int.Parse(mb[i].Groups[2].Value);
                    if (ret != 0)
                        return ret;
                }

                return 0;
            });
        }
*/
func NumericalSort(ar []string) ([]string, error) {
	resultMap := make(map[int]string, 0)

	for _, v := range ar {
		numStr := strings.Replace(v, ".png", "", -1)
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return nil, err
		}

		resultMap[num] = v
	}

	result := make([]string, 0)
	for i := 0; i < len(resultMap); i++ {
		result = append(result, resultMap[i])
	}

	return result, nil
}
