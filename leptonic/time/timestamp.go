/*
   Copyright Farcloser.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package time // import "github.com/containerd/nerdctl/v2/leptonic/time"

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidArgument = errors.New("invalid argument")
)

// These are additional predefined layouts for use in Time.Format and Time.Parse
// with --since and --until parameters for `docker logs` and `docker events`
const (
	rFC3339Local     = "2006-01-02T15:04:05"           // RFC3339 with local timezone
	rFC3339NanoLocal = "2006-01-02T15:04:05.999999999" // RFC3339Nano with local timezone
	dateWithZone     = "2006-01-02Z07:00"              // RFC3339 with time at 00:00:00
	dateLocal        = "2006-01-02"                    // RFC3339 with local timezone and time at 00:00:00
)

// GetTimestamp tries to parse given string as golang duration,
// then RFC3339 time and finally as a Unix timestamp. If
// any of these were successful, it returns a Unix timestamp
// as string otherwise returns the given value back.
// In case of duration input, the returned timestamp is computed
// as the given reference time minus the amount of the duration.
func GetTimestamp(value string, reference time.Time) (string, error) {
	if d, err := time.ParseDuration(value); value != "0" && err == nil {
		return strconv.FormatInt(reference.Add(-d).Unix(), 10), nil
	}

	var format string
	// if the string has a Z or a + or three dashes use parse otherwise use parseinlocation
	parseInLocation := !(strings.ContainsAny(value, "zZ+") || strings.Count(value, "-") == 3)

	if strings.Contains(value, ".") {
		if parseInLocation {
			format = rFC3339NanoLocal
		} else {
			format = time.RFC3339Nano
		}
	} else if strings.Contains(value, "T") {
		// we want the number of colons in the T portion of the timestamp
		tcolons := strings.Count(value, ":")
		// if parseInLocation is off and we have a +/- zone offset (not Z) then
		// there will be an extra colon in the input for the tz offset subtract that
		// colon from the tcolons count
		if !parseInLocation && !strings.ContainsAny(value, "zZ") && tcolons > 0 {
			tcolons--
		}
		if parseInLocation {
			switch tcolons {
			case 0:
				format = "2006-01-02T15"
			case 1:
				format = "2006-01-02T15:04"
			default:
				format = rFC3339Local
			}
		} else {
			switch tcolons {
			case 0:
				format = "2006-01-02T15Z07:00"
			case 1:
				format = "2006-01-02T15:04Z07:00"
			default:
				format = time.RFC3339
			}
		}
	} else if parseInLocation {
		format = dateLocal
	} else {
		format = dateWithZone
	}

	var t time.Time
	var err error

	if parseInLocation {
		t, err = time.ParseInLocation(format, value, time.FixedZone(reference.Zone()))
	} else {
		t, err = time.Parse(format, value)
	}

	if err != nil {
		// if there is a `-` then it's an RFC3339 like timestamp
		if strings.Contains(value, "-") {
			return "", err // was probably an RFC3339 like timestamp but the parser failed with an error
		}
		if _, _, err := parseTimestamp(value); err != nil {
			return "", fmt.Errorf("%w: %q", ErrInvalidArgument, value)
		}
		return value, nil // unix timestamp in and out case (meaning: the value passed at the command line is already in the right format for passing to the server)
	}

	return fmt.Sprintf("%d.%09d", t.Unix(), int64(t.Nanosecond())), nil
}

func parseTimestamp(value string) (sec int64, nsec int64, err error) {
	s, n, ok := strings.Cut(value, ".")
	sec, err = strconv.ParseInt(s, 10, 64)
	if err != nil {
		return sec, 0, err
	}
	if !ok {
		return sec, 0, nil
	}
	nsec, err = strconv.ParseInt(n, 10, 64)
	if err != nil {
		return sec, nsec, err
	}
	// should already be in nanoseconds but just in case convert n to nanoseconds
	nsec = int64(float64(nsec) * math.Pow(float64(10), float64(9-len(n))))
	return sec, nsec, nil
}
