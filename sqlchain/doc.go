/*
 * Copyright 2018 The ThunderDB Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package sqlchain provides a blockchain implementation for database state tracking.
//
// Bucket hierarchical structure is as following:
//
// [ root ]
//    |
//    +-- [ sql-chain meta ]
//    |      |
//    |      +-- [ state ]: State
//    |      |
//    |      +-- [ block ]
//    |      |       |
//    |      |       +--- [ hash1 ]: Block1
//    |      |       +--- [ hash2 ]: Block2
//    |      |       +--- [ hash3 ]: Block3
//    |      |       |
//    |      |       +--- ...
//    |      |
//    |      +-- [ query ]
//    |              |
//    |              +--- ...
//    |              |
//    |              +--- [ height 7 ]
//    |              |         |
//    |              |         +--- [ request ]
//    |              |         |        |
//    |              |         |        +--- [ header hash1 ]: Request1
//    |              |         |        +--- [ header hash2 ]: Request2
//    |              |         |        +--- [ header hash3 ]: Request3
//    |              |         |        |
//    |              |         |        +--- ...
//    |              |         |
//    |              |         +--- [ response ]
//    |              |         |        |
//    |              |         |        +--- [ header hash1 ]: Response1
//    |              |         |        +--- [ header hash2 ]: Response2
//    |              |         |        +--- [ header hash3 ]: Response3
//    |              |         |        |
//    |              |         |        +--- ...
//    |              |         |
//    |              |         +--- [ ack ]
//    |              |                  |
//    |              |                  +--- [ header hash1 ]: Ack1
//    |              |                  +--- [ header hash2 ]: Ack2
//    |              |                  +--- [ header hash3 ]: Ack3
//    |              |                  |
//    |              |                  +--- ...
//    |              |
//    |              |
//    |              +--- [ height 8 ]
//    |              |         |
//    |              |         +--- [ request ]
//    |              |         |        |
//    |              |         |        +--- ...
//    |              |         |
//    |              |         +--- [ response ]
//    |              |         |        |
//    |              |         |        +--- ...
//    |              |         |
//    |              |         +--- [ ack ]
//    |              |                  |
//    |              |                  +--- ...
//    |              |
//    |              +--- [ height 9 ]
//    |              |         |
//    |              |         +--- [ request ]
//    |              |         |        |
//    |              |         |        +--- ...
//    |              |         |
//    |              |         +--- [ response ]
//    |              |         |        |
//    |              |         |        +--- ...
//    |              |         |
//    |              |         +--- [ ack ]
//    |              |                  |
//    |              |                  +--- ...
//    |              |
//    |              +--- ...
//    |
//    |-- [ other bucket ]
//    |-- [ other bucket ]
//    |-- [ other bucket ]
//    |
//    +-- ...
//
package sqlchain
