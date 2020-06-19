// Copyright 2019 The go-relianz Authors
// This file is part of the go-relianz library.
//
// The go-relianz library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-relianz library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-relianz library. If not, see <http://www.gnu.org/licenses/>.

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/relianz2019/relianz/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("dsp/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("dsp/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("dsp/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("dsp/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("dsp/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("dsp/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("dsp/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("dsp/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("dsp/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("dsp/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("dsp/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("dsp/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("dsp/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("dsp/downloader/states/drop", nil)
)
