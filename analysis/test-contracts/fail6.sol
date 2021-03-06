pragma solidity ^0.4.0;

contract Fail6 {
    function foo(bool a1) public pure returns (int) {
        int x = 0;
        if (a1) {
            x += 1;
        }
        assert(a1);
    }
}

/*
606060405260043610603f576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806345557578146044575b600080fd5b3415604e57600080fd5b606460048080351515906020019091905050607a565b6040518082815260200191505060405180910390f35b600080600090508215608d576001810190505b821515609557fe5b509190505600a165627a7a723058209f2f9f23a227d517776de1e8c286e5e6f06e2e42da2d223de7602810d0f1b32b00
0      PUSH1  => 96
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
54     PUSH4  => 1163228536
59     EQ
60     PUSH1  => 68
62     JUMPI
63     JUMPDEST
64     PUSH1  => 0
66     DUP1
67     REVERT
68     JUMPDEST
69     CALLVALUE
70     ISZERO
71     PUSH1  => 78
73     JUMPI
74     PUSH1  => 0
76     DUP1
77     REVERT
78     JUMPDEST
79     PUSH1  => 100
81     PUSH1  => 4
83     DUP1
84     DUP1
85     CALLDATALOAD
86     ISZERO
87     ISZERO
88     SWAP1
89     PUSH1  => 32
91     ADD
92     SWAP1
93     SWAP2
94     SWAP1
95     POP
96     POP
97     PUSH1  => 122
99     JUMP
100    JUMPDEST
101    PUSH1  => 64
103    MLOAD
104    DUP1
105    DUP3
106    DUP2
107    MSTORE
108    PUSH1  => 32
110    ADD
111    SWAP2
112    POP
113    POP
114    PUSH1  => 64
116    MLOAD
117    DUP1
118    SWAP2
119    SUB
120    SWAP1
121    RETURN
122    JUMPDEST
123    PUSH1  => 0
125    DUP1
126    PUSH1  => 0
128    SWAP1
129    POP
130    DUP3
131    ISZERO
132    PUSH1  => 141
134    JUMPI
135    PUSH1  => 1
137    DUP2
138    ADD
139    SWAP1
140    POP
141    JUMPDEST
142    DUP3
143    ISZERO
144    ISZERO
145    PUSH1  => 149
147    JUMPI
148    Missing opcode 0xfe
149    JUMPDEST
150    POP
151    SWAP2
152    SWAP1
153    POP
154    JUMP
155    STOP
156    LOG1
157    PUSH6  => 108278179835992
164    SHA3
165    SWAP16
166    Missing opcode 0x2f
167    SWAP16
168    Missing opcode 0x23
169    LOG2
170    Missing opcode 0x27
171    Missing opcode 0xd5
172    OR
173    PUSH24  => 9223372036854775807
*/
