---
sidebar_position: 1
---

# 튜토리얼 소개

**5분 안에 Docusaurus를 배워봅시다**.

## 시작하기

**새 사이트를 만들어서** 시작하세요.

또는 **[docusaurus.new](https://docusaurus.new)**에서 **Docusaurus를 바로 체험**해보세요.

### 필요한 것들

- [Node.js](https://nodejs.org/en/download/) 버전 18.0 이상:
  - Node.js를 설치할 때 의존성 관련 체크박스를 모두 선택하는 것을 권장합니다.

## 새 사이트 생성

**클래식 템플릿**을 사용하여 새 Docusaurus 사이트를 생성하세요.

명령어를 실행하면 클래식 템플릿이 자동으로 프로젝트에 추가됩니다:

```bash
npm init docusaurus@latest my-website classic
```

이 명령어는 명령 프롬프트, Powershell, 터미널 또는 코드 에디터의 통합 터미널에 입력할 수 있습니다.

이 명령어는 Docusaurus를 실행하는 데 필요한 모든 의존성도 함께 설치합니다.

## 사이트 실행

개발 서버를 실행하세요:

```bash
cd my-website
npm run start
```

`cd` 명령어는 작업 중인 디렉토리를 변경합니다. 새로 생성된 Docusaurus 사이트를 작업하려면 터미널에서 해당 위치로 이동해야 합니다.

`npm run start` 명령어는 웹사이트를 로컬에서 빌드하고 개발 서버를 통해 제공하며, http://localhost:3000/에서 확인할 수 있습니다.

`docs/intro.md` (이 페이지)를 열어 몇 줄을 수정해보세요: 사이트가 **자동으로 새로고침**되어 변경사항을 표시합니다.