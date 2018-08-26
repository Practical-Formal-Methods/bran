pragma solidity ^0.4.0;

contract Pass8 {
    function add() public pure returns (int) {
        address a;
        address b;
        assert(a == b); // equal because of equal default values
    }
}

/*
608060405260043610603f576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680634f2be91f146044575b600080fd5b348015604f57600080fd5b506056606c565b6040518082815260200191505060405180910390f35b60008060008073ffffffffffffffffffffffffffffffffffffffff168273ffffffffffffffffffffffffffffffffffffffff1614151560a757fe5b5050905600a165627a7a723058202beda531f149a609eb4db500f7f6c6fc587abc10bdcde3f4638c0d754a7428d000
0      PUSH1  => 128
2      PUSH1  => 64
4      MSTORE
5      PUSH1  => 4
7      CALLDATASIZE
8      LT
9      PUSH1  => 63
11     JUMPI
12     PUSH1  => 0
14     CALLDATALOAD
15     PUSH29  => 9223372036854775807
45     SWAP1
46     DIV
47     PUSH4  => 4294967295
52     AND
53     DUP1
54     PUSH4  => 1328277791
59     EQ
60     PUSH1  => 68
62     JUMPI
63     JUMPDEST
64     PUSH1  => 0
66     DUP1
67     REVERT
68     JUMPDEST
69     CALLVALUE
70     DUP1
71     ISZERO
72     PUSH1  => 79
74     JUMPI
75     PUSH1  => 0
77     DUP1
78     REVERT
79     JUMPDEST
80     POP
81     PUSH1  => 86
83     PUSH1  => 108
85     JUMP
86     JUMPDEST
87     PUSH1  => 64
89     MLOAD
90     DUP1
91     DUP3
92     DUP2
93     MSTORE
94     PUSH1  => 32
96     ADD
97     SWAP2
98     POP
99     POP
100    PUSH1  => 64
102    MLOAD
103    DUP1
104    SWAP2
105    SUB
106    SWAP1
107    RETURN
108    JUMPDEST
109    PUSH1  => 0
111    DUP1
112    PUSH1  => 0
114    DUP1
115    PUSH20  => 9223372036854775807
136    AND
137    DUP3
138    PUSH20  => 9223372036854775807
159    AND
160    EQ
161    ISZERO
162    ISZERO
163    PUSH1  => 167
165    JUMPI
166    Missing opcode 0xfe
167    JUMPDEST
168    POP
169    POP
170    SWAP1
171    JUMP
172    STOP
173    LOG1
174    PUSH6  => 108278179835992
181    SHA3
182    Missing opcode 0x2b
183    Missing opcode 0xed
184    Missing opcode 0xa5
185    BALANCE
186    CALL
187    Missing opcode 0x49
188    Missing opcode 0xa6
189    MULMOD
190    Missing opcode 0xeb
191    Missing opcode 0x4d
192    Missing opcode 0xb5
193    STOP
194    Missing opcode 0xf7
195    Missing opcode 0xf6
196    Missing opcode 0xc6
197    Missing opcode 0xfc
198    PC
199    PUSH27  => 9223372036854775807
*/
