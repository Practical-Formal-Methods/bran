pragma solidity ^0.4.0;

contract Fail1 {
    function add(int a, int b) public pure returns (int) {
        a = 10;
        b = 20;
        if (a != b) {
            assert(false);
        }
    }
}

/*
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
54     PUSH4  => 2784215611
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
79     PUSH1  => 107
81     PUSH1  => 4
83     DUP1
84     DUP1
85     CALLDATALOAD
86     SWAP1
87     PUSH1  => 32
89     ADD
90     SWAP1
91     SWAP2
92     SWAP1
93     DUP1
94     CALLDATALOAD
95     SWAP1
96     PUSH1  => 32
98     ADD
99     SWAP1
100    SWAP2
101    SWAP1
102    POP
103    POP
104    PUSH1  => 129
106    JUMP
107    JUMPDEST
108    PUSH1  => 64
110    MLOAD
111    DUP1
112    DUP3
113    DUP2
114    MSTORE
115    PUSH1  => 32
117    ADD
118    SWAP2
119    POP
120    POP
121    PUSH1  => 64
123    MLOAD
124    DUP1
125    SWAP2
126    SUB
127    SWAP1
128    RETURN
129    JUMPDEST
130    PUSH1  => 0
132    PUSH1  => 10
135    POP
136    PUSH1  => 20
138    SWAP2
139    POP
140    DUP2
141    DUP4
142    EQ
143    ISZERO
144    ISZERO
145    PUSH1  => 157
147    JUMPI
148    PUSH1  => 0
150    ISZERO
151    ISZERO
152    PUSH1  => 156
154    JUMPI
155    Missing opcode 0xfe
156    JUMPDEST
157    JUMPDEST
158    SWAP3
159    SWAP2
160    POP
161    POP
162    JUMP
163    STOP
164    LOG1
165    PUSH6  => 108278179835992
172    SHA3
173    PUSH
174    PUSH25  => 9223372036854775807
200    EXP
201    LOG3
202    SWAP16
203    Missing opcode 0xe2
204    CALLCODE
205    STOP
*/