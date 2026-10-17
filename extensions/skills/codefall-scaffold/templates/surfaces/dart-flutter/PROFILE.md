# Surface profile: dart-flutter

**Status:** supported.
**ADR prefix:** `DART` — this profile's ADRs are `ADR-DART-01` upward.

Flutter apps, on mobile, desktop, and the web. Covers the app as one surface whatever it talks to.

## Fits when

- A Flutter app on iOS or Android, **or**
- A Flutter app on macOS, Windows, or Linux, **or**
- A Flutter web build of either, shipped from the same package.

The app does not care what is on the other end of its gateways. Per ADR-BASE-01 a gateway is "an
interface to any external system" — an HTTP API, a platform channel, a local database, and a
WebSocket are all the same shape to a use case. A Flutter app that renders and calls an API is this
profile; so is one that decides, validates, and reconciles offline. The second needs the seam step
in `SKILL.md`, because then the domain lives on both sides.

### Platform code

Kotlin, Swift, C++, or Dart FFI under `android/`, `ios/`, `macos/`, `windows/`, and `linux/` that
is stock boilerplate plus a few channels wrapping OS APIs — the camera, secure storage,
notifications, a file dialog — is **gateway implementation**, and the app is entirely this profile.
The channel's Dart side lives in the `infrastructure/` ring of the component that needs it, behind
an interface its `application/` layer owns. A use case never imports `MethodChannel`.

Platform code that carries **real domain logic** — heavy compute, a native SDK with rules of its
own, security-sensitive work kept out of Dart — is a second surface, and its profile
(`kotlin-native`, `swift-native`) is planned, not supported. The project stops there, per the stack
question in `SKILL.md`. The test is *where domain logic lives*, and it is a question for the user:
a fresh `flutter create` ships every platform directory with code in it.

## Does not fit

Non-Flutter Dart. A Dart server, a Dart CLI, or a package with no Flutter dependency is a different
surface with a different presentation ring, and no profile covers it yet.

Platform code that holds domain logic, as above.

## Visibility model

Dart has **library-private names and nothing else**. An identifier that starts with `_` is private
to its library; everything else is public to the whole package, and there is no package-private
level. The `implementation_imports` lint guards one package's `src/` from another package, and is
silent inside a package. So, as in TypeScript, a component's facade file is a convention the
compiler will not enforce, and ADR-DART-03 is built around a linter — `import_lint` — for both the
facade rule and the layer rule. The facade half can be moved onto the compiler by making each
component its own package in a pub workspace; the profile names that as the upgrade for a
repository with several apps, not the default for one.

## Layout

```
lib/
  main.dart                      composition root: configureDependencies(), ProviderScope, runApp
  main_<env>.dart                one per extra environment, differing only in the environment name
  app/
    injection.dart               @InjectableInit, configureDependencies
    injection.config.dart        generated, committed
    router.dart                  go_router configuration, built from component facades
  components/<name>/
    <name>.dart                  the facade: `export ... show ...` and nothing else
    src/
      domain/                    entities, value objects, errors
      application/               use cases + the interfaces they need
      infrastructure/            gateway implementations: retrofit services, channels, stores
      presentation/              screens, widgets, providers, notifiers, bridge providers
  shared/<module>/               focused shared modules, each a facade over its own src/
```

## Toolchain

| Concern | Choice | ADR |
| --- | --- | --- |
| Dependency injection | `get_it` registered by `injectable`; `GetIt.I` in two places only | ADR-DART-01 |
| Interface granularity | One per gateway role per **component**, not per use case | ADR-DART-01 |
| App state | Riverpod 3 — async providers, `Notifier`, generator and lint on | ADR-DART-02 |
| Navigation | go_router | ADR-DART-02 |
| Local widget state | `setState`, or `flutter_hooks` | ADR-DART-02 |
| API client | `retrofit` on `dio`; `swagger_parser` writes the service when an OpenAPI document exists; `dio` directly only where `retrofit` cannot express the call | — |
| Models | `freezed` with `json_serializable`; a primary constructor when no union, `copyWith`, or JSON is needed | — |
| Code generation | `build_runner`, one command for every generator; generated files committed | — |
| Boundary enforcement | `import_lint`, severity `error`, generated per project | ADR-DART-03 |
| Lints | `flutter_lints`, `riverpod_lint` | ADR-DART-03 |
| Formatting | `dart format` | — |
| Tests | `flutter_test`, hand-written fakes, bridge-provider overrides | ADR-DART-01 |
| End-to-end runner | none in this version; `codefall-equip` refuses the `spec` modality for Flutter | — |

The API client is a toolchain default rather than an ADR because the gateway interface already
contains it: swapping `retrofit` for a hand-written `dio` call touches one class in
`infrastructure/` and nothing inward. Chopper remains a reasonable generator on `package:http` for a
team already on it, and is not the default here because the `dio` ecosystem is several times
larger, which is what agents draw on.

**State management is the one override this profile offers.** The interview says *we will use
Riverpod for state management unless you want to override that decision now*, proposes nothing
else, and an override amends ADR-DART-02 for the project. ADR-DART-01's bridge changes shape with
it; nothing inward does.

## ADRs this profile supplies

- `adrs/ADR-DART-01-dependency-injection.md`
- `adrs/ADR-DART-02-app-state.md`
- `adrs/ADR-DART-03-boundary-enforcement.md`

Shared ADR-BASE-01 and ADR-BASE-02 come from `templates/adrs/` and apply here. **ADR-BASE-03 is
dropped**: a Flutter app is a surface that could never be split into services, which is the one
case that gate names. The skeleton carries no link to it and no extraction-readiness section.

## Depth notes

Beyond docs, this profile's **project files** tier owes: `pubspec.yaml`, `analysis_options.yaml`
carrying `flutter_lints`, `riverpod_lint`, and the generated `import_lint` rules (ADR-DART-03),
`build.yaml` when a generator needs options, and CI running `flutter pub get`,
`dart run build_runner build --delete-conflicting-outputs` followed by `git diff --exit-code`,
`flutter analyze`, `dart run import_lint`, and `flutter test`.

**Add every dependency with the package manager** — `flutter pub add <name>`,
`flutter pub add --dev <name>` — never by writing a version into `pubspec.yaml` by hand. The
package manager resolves the current release and writes the lockfile to match it; a version written
from memory is whatever was current when the model was trained. The toolchain needs the current
stable Flutter: `import_lint` wants Dart 3.10 or later and primary constructors arrived in 3.13.

**`analysis_options.yaml` is generated, not copied.** The facade rules name each component, so the
file cannot exist until the component names and the package name are known. Emit the wildcarded
layer rules plus one facade rule per component; ADR-DART-03 has the shape.

The local scripts at either code tier: `update` runs `flutter pub get` and then the `build_runner`
command above, and `mise install` first when the project pins a toolchain; `start` exits `0` with
nothing to bring up. They are two scripts under `tool/`, `tool/local-update.sh` and
`tool/local-start.sh`, declared by those paths.

The **runnable skeleton** tier adds `lib/main.dart` and `lib/app/` wiring the composition root end
to end with no features in it — `configureDependencies`, a `ProviderScope`, a go_router with one
route — plus each component directory with its facade file and the four nested layer folders.

Verify the boundary rules the way step 6 requires, and note that this profile's rules fail in one
place: a deliberate cross-component reach-around and a deliberate outward layer import must **each**
fail `dart run import_lint` while `flutter analyze` is otherwise clean. Check both, because a rule
whose `target` glob matches nothing reports nothing and exits `0`. Resolve one use case from the
container in a smoke test as well: a binding left out of the graph fails at first resolution and
nowhere earlier.
