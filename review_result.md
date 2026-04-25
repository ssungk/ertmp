# AMF 패키지 리뷰 결과 정리

반복 리뷰에서 동일한 지적이 반복되는 것을 방지하기 위해 검토 결과를 기록한다.
각 항목은 **조치 완료**, **조치 불필요 (이유)**, **TODO** 중 하나로 분류한다.

---

## 조치 불필요

### `encodeProperties` 키 정렬 제거

**지적 내용**: `encodeProperties`에서 `sort.Strings(keys)`로 키를 정렬하면 결정론적 출력을 보장하지만, 매 호출마다 keys slice 할당이 발생한다.

**결론**: 정렬 제거, 직접 map 순회로 변경.

**이유**: AMF0 스펙은 Object/ECMAArray의 키 순서를 정의하지 않는다. 수신측도 키 이름으로 값을 찾으므로 순서가 달라도 동작에 영향 없다. 바이트 비교 테스트는 단일 키 케이스만 사용하므로 영향 없고, 다수 키 테스트는 이미 round-trip 방식이다. 순서 보장의 필요가 없으므로 정렬은 과잉이다.

---

### `decodeECMAArray` 암묵적 타입 변환 (`return m` vs `return ECMAArray(m)`)

**지적 내용**: `decodeECMAArray`에서 `return m, nil`이 `map[string]any`를 `ECMAArray`로 암묵적 변환하는데, `return ECMAArray(m), nil`처럼 명시적 변환이 더 낫다는 지적.

**결론**: 조치 불필요. 현 상태(`return m, nil`) 유지.

**이유**: `ECMAArray`와 `map[string]any`는 동일한 underlying type이므로 Go 스펙상 암묵적 변환이 보장된다. 린터도 명시적 변환을 redundant로 표시하며, IDE에서 회색 글자로 노이즈가 된다. 동작 차이 없음.

---

### `encodeProperties` 종료 마커 slice literal 스타일 불일치

**지적 내용**: `e.buf.Write([]byte{0x00, 0x00, objectEndMarker})`가 파일 내 다른 `var b [N]byte` 패턴과 스타일이 다르다.

**결론**: 조치 불필요.

**이유**: 3바이트 리터럴 수준에서 일관성 강제는 오버엔지니어링이다. 컴파일러가 최적화하므로 성능 차이도 없다. `var b [3]byte / b[2] = objectEndMarker`로 바꾸면 오히려 0x00 두 개가 암묵적으로 zero-init에 의존한다는 점이 숨겨져 가독성이 떨어진다.

---

### 재귀 깊이 제한 없음

**지적 내용**: `decodeAMF0` ↔ `decodeObject` / `decodeStrictArray`가 상호 재귀하며 깊이 제한이 없어 스택 오버플로우 가능.

**결론**: 조치 불필요.

**이유**: 이 디코더는 스트리밍 방식으로 읽는 것이 아니라, 호출자가 유한한 바이트 슬라이스를 `Decode(data []byte)`에 전달하는 구조다. 재귀가 한 단계 깊어지려면 최소 약 7바이트(object marker + 최단 키 + end marker)가 소비된다. 따라서 재귀 깊이는 입력 크기에 의해 자연히 제한되며, 무한 재귀는 불가능하다. RTMP 메시지는 통상 수 KB 이하이므로 스택 소진 위험은 현실적이지 않다. transport 레이어에서 메시지 크기를 제한하면 재귀 깊이도 함께 제한된다.
