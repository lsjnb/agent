//go:build darwin && arm64

package gpu

import (
	"context"
	"fmt"
	"strings"
	"unsafe"

	"github.com/ebitengine/purego"

	"github.com/nezhahq/agent/pkg/util"
)

type (
	CFStringEncoding = uint32
	CFIndex          = int32
	CFTypeID         = int32
	CFNumberType     = CFIndex
	CFTypeRef        = unsafe.Pointer
	CFStringRef      = unsafe.Pointer
	CFDictionaryRef  = unsafe.Pointer

	machPort        = uint32
	ioIterator      = uint32
	ioObject        = uint32
	ioRegistryEntry = uint32
	ioService       = uint32
	IOOptionBits    = uint32
)

type (
	CFStringCreateWithCStringFunc = func(alloc uintptr, cStr string, encoding CFStringEncoding) CFStringRef
	CFGetTypeIDFunc               = func(cf uintptr) CFTypeID
	CFStringGetTypeIDFunc         = func() CFTypeID
	CFStringGetLengthFunc         = func(theString uintptr) int32
	CFStringGetCStringFunc        = func(cfStr uintptr, buffer *byte, size CFIndex, encoding CFStringEncoding) bool
	CFDictionaryGetTypeIDFunc     = func() CFTypeID
	CFDictionaryGetValueFunc      = func(dict, key uintptr) unsafe.Pointer
	CFNumberGetValueFunc          = func(number uintptr, theType CFNumberType, valuePtr uintptr) bool
	CFReleaseFunc                 = func(cf uintptr)

	IOServiceGetMatchingServicesFunc    = func(mainPort machPort, matching uintptr, existing *ioIterator) ioService
	IOIteratorNextFunc                  = func(iterator ioIterator) ioObject
	IOServiceMatchingFunc               = func(name string) CFDictionaryRef
	IORegistryEntrySearchCFPropertyFunc = func(entry ioRegistryEntry, plane string, key, allocator uintptr, options IOOptionBits) CFTypeRef
	IOObjectReleaseFunc                 = func(object ioObject) int
)

const (
	KERN_SUCCESS   = 0
	MACH_PORT_NULL = 0
	IOSERVICE_GPU  = "IOAccelerator"

	kIOServicePlane               = "IOService"
	kIORegistryIterateRecursively = 1
	kCFStringEncodingUTF8         = 0x08000100
	kCFNumberIntType              = 9
)

var (
	kCFAllocatorDefault uintptr  = 0
	kIOMainPortDefault  machPort = 0
)

var (
	coreFoundation, _ = purego.Dlopen("/System/Library/Frameworks/CoreFoundation.framework/CoreFoundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	ioKit, _          = purego.Dlopen("/System/Library/Frameworks/IOKit.framework/IOKit", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
)

var (
	CFStringCreateWithCString CFStringCreateWithCStringFunc
	CFGetTypeID               CFGetTypeIDFunc
	CFStringGetTypeID         CFStringGetTypeIDFunc
	CFStringGetLength         CFStringGetLengthFunc
	CFStringGetCString        CFStringGetCStringFunc
	CFDictionaryGetTypeID     CFDictionaryGetTypeIDFunc
	CFDictionaryGetValue      CFDictionaryGetValueFunc
	CFNumberGetValue          CFNumberGetValueFunc
	CFRelease                 CFReleaseFunc

	IOServiceGetMatchingServices    IOServiceGetMatchingServicesFunc
	IOIteratorNext                  IOIteratorNextFunc
	IOServiceMatching               IOServiceMatchingFunc
	IORegistryEntrySearchCFProperty IORegistryEntrySearchCFPropertyFunc
	IOObjectRelease                 IOObjectReleaseFunc
)

func init() {
	purego.RegisterLibFunc(&CFStringCreateWithCString, coreFoundation, "CFStringCreateWithCString")
	purego.RegisterLibFunc(&CFGetTypeID, coreFoundation, "CFGetTypeID")
	purego.RegisterLibFunc(&CFStringGetTypeID, coreFoundation, "CFStringGetTypeID")
	purego.RegisterLibFunc(&CFStringGetLength, coreFoundation, "CFStringGetLength")
	purego.RegisterLibFunc(&CFStringGetCString, coreFoundation, "CFStringGetCString")
	purego.RegisterLibFunc(&CFDictionaryGetTypeID, coreFoundation, "CFDictionaryGetTypeID")
	purego.RegisterLibFunc(&CFDictionaryGetValue, coreFoundation, "CFDictionaryGetValue")
	purego.RegisterLibFunc(&CFNumberGetValue, coreFoundation, "CFNumberGetValue")
	purego.RegisterLibFunc(&CFRelease, coreFoundation, "CFRelease")

	purego.RegisterLibFunc(&IOServiceGetMatchingServices, ioKit, "IOServiceGetMatchingServices")
	purego.RegisterLibFunc(&IOIteratorNext, ioKit, "IOIteratorNext")
	purego.RegisterLibFunc(&IOServiceMatching, ioKit, "IOServiceMatching")
	purego.RegisterLibFunc(&IORegistryEntrySearchCFProperty, ioKit, "IORegistryEntrySearchCFProperty")
	purego.RegisterLibFunc(&IOObjectRelease, ioKit, "IOObjectRelease")
}

func GetHost(_ context.Context) ([]string, error) {
	models, err := findDevices("model")
	if err != nil {
		return nil, err
	}
	return util.RemoveDuplicate(models), nil
}

func GetState(_ context.Context) ([]float64, error) {
	usage, err := findUtilization("PerformanceStatistics", "Device Utilization %")
	return []float64{float64(usage)}, err
}

func findDevices(key string) ([]string, error) {
	var iterator ioIterator
	var results []string

	iv := IOServiceGetMatchingServices(kIOMainPortDefault, uintptr(IOServiceMatching(IOSERVICE_GPU)), &iterator)
	if iv != KERN_SUCCESS {
		return nil, fmt.Errorf("error retrieving GPU entry")
	}

	var service ioObject
	index := 0

	for {
		service = IOIteratorNext(iterator)
		if service == MACH_PORT_NULL {
			break
		}

		cfStr := CFStringCreateWithCString(kCFAllocatorDefault, key, kCFStringEncodingUTF8)
		r, _ := findProperties(service, uintptr(cfStr), 0)
		result, _ := r.(string)
		IOObjectRelease(service)

		if strings.Contains(result, "Apple") {
			results = append(results, result)
			index++
		}
	}

	IOObjectRelease(iterator)
	return results, nil
}

func findUtilization(key, dictKey string) (int, error) {
	var iterator ioIterator
	var result int
	var err error

	iv := IOServiceGetMatchingServices(kIOMainPortDefault, uintptr(IOServiceMatching(IOSERVICE_GPU)), &iterator)
	if iv != KERN_SUCCESS {
		return 0, fmt.Errorf("error retrieving GPU entry")
	}

	// Only retrieving the utilization of a single GPU here
	var service ioObject
	for {
		service = IOIteratorNext(iterator)
		if service == MACH_PORT_NULL {
			break
		}

		cfStr := CFStringCreateWithCString(kCFAllocatorDefault, key, CFStringEncoding(kCFStringEncodingUTF8))
		cfDictStr := CFStringCreateWithCString(kCFAllocatorDefault, dictKey, CFStringEncoding(kCFStringEncodingUTF8))

		r, rerr := findProperties(service, uintptr(cfStr), uintptr(cfDictStr))
		result, _ = r.(int)

		CFRelease(uintptr(cfStr))
		CFRelease(uintptr(cfDictStr))

		if rerr != nil {
			err = rerr
			IOObjectRelease(service)
			continue
		} else if result != 0 {
			break
		}
	}

	IOObjectRelease(service)
	IOObjectRelease(iterator)

	return result, err
}

func findProperties(service ioRegistryEntry, key, dictKey uintptr) (any, error) {
	properties := IORegistryEntrySearchCFProperty(service, kIOServicePlane, key, kCFAllocatorDefault, kIORegistryIterateRecursively)
	ptrValue := uintptr(properties)
	if properties != nil {
		switch CFGetTypeID(ptrValue) {
		// model
		case CFStringGetTypeID():
			length := CFStringGetLength(ptrValue) + 1 // null terminator
			buf := make([]byte, length-1)
			CFStringGetCString(ptrValue, &buf[0], length, uint32(kCFStringEncodingUTF8))
			CFRelease(ptrValue)
			return string(buf), nil
		// PerformanceStatistics
		case CFDictionaryGetTypeID():
			cfValue := CFDictionaryGetValue(ptrValue, dictKey)
			if cfValue != nil {
				var value int
				if CFNumberGetValue(uintptr(cfValue), kCFNumberIntType, uintptr(unsafe.Pointer(&value))) {
					return value, nil
				} else {
					return nil, fmt.Errorf("failed to exec CFNumberGetValue")
				}
			} else {
				return nil, fmt.Errorf("failed to exec CFDictionaryGetValue")
			}
		}
	}
	return nil, fmt.Errorf("failed to exec IORegistryEntrySearchCFProperty")
}
