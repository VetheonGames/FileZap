package vpn

import "net"

// maskBits returns the number of bits in a netmask
func maskBits(mask net.IPMask) int {
    bits := 0
    for _, b := range mask {
        bits += hammingWeight(b)
    }
    return bits
}

// hammingWeight counts the number of bits set to 1 in a byte
func hammingWeight(b byte) int {
    count := 0
    for b != 0 {
        count += int(b & 1)
        b >>= 1
    }
    return count
}
