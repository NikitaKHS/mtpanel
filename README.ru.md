# MTPanel (RU)

Self-hosted веб-панель для управления **TeleMT** на Linux-сервере.

## Содержание

1. [Что это](#что-это)
2. [Возможности](#возможности)
3. [Требования](#требования)
4. [Установка в одну строку](#установка-в-одну-строку)
5. [Параметры установщика](#параметры-установщика)
6. [Что установится автоматически](#что-установится-автоматически)
7. [Первый запуск](#первый-запуск)
8. [Скриншоты](#скриншоты)
9. [Полное удаление](#полное-удаление)
10. [Диагностика](#диагностика)

## Что это

MTPanel — это панель управления TeleMT с простым сценарием:

- поставить одной командой;
- открыть UI;
- установить/перезапустить TeleMT;
- генерировать и отзывать proxy-ссылки.

## Возможности

- Авторизация и первичная настройка.
- Управление TeleMT (`install/start/stop/restart`).
- Генерация proxy-ссылок (`tg://proxy?...`) в 1 клик.
- Просмотр логов и статуса.
- Проверка и применение обновлений TeleMT.
- Автоматическая настройка firewall в install script.

## Требования

- Linux (Ubuntu/Debian/RHEL-family/Arch-family).
- `systemd`.
- Root-доступ (`sudo`).

## Установка в одну строку

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash
```

С явными параметрами:

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash -s -- --port 8080 --proxy-port 443
```

## Параметры установщика

- `--port <port>` — порт панели (по умолчанию `8080`).
- `--proxy-port <port>` — порт TeleMT (по умолчанию `443`).
- `--panel-allow <CIDR>` — доступ к панели только из указанной сети (например, `1.2.3.4/32`).
- `--repo <owner/repo>` — репозиторий для релизов.

Пример с ограничением доступа к панели:

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash -s -- --panel-allow 1.2.3.4/32
```

## Что установится автоматически

Установщик:

1. проверит систему и зависимости;
2. скачает/соберет бинарник и фронтенд;
3. создаст конфиг в `/etc/mtpanel/config.json`;
4. удалит старые `mtpanel.service.d` overrides;
5. поставит стабильный `mtpanel.service`;
6. применит firewall:
   - `proxy-port` открыт для всех;
   - `panel-port` ограничен SSH IP (или открыт для всех, если IP не определён).

## Первый запуск

1. Откройте `http://<SERVER_IP>:8080`.
2. Перейдите на `/setup`.
3. Задайте пароль администратора.
4. Войдите через `/login`.
5. Откройте раздел `Прокси` и установите TeleMT.

## Скриншоты

Добавьте изображения в `docs/screenshots/`:

```text
docs/screenshots/dashboard.png
docs/screenshots/proxy.png
docs/screenshots/links.png
docs/screenshots/updates.png
```

Пример вставки:

```md
![Dashboard](docs/screenshots/dashboard.png)
```

## Полное удаление

```bash
sudo systemctl stop mtpanel 2>/dev/null || true
sudo systemctl disable mtpanel 2>/dev/null || true
sudo pkill -f '/opt/mtpanel/mtpanel' || true

sudo systemctl stop telemt 2>/dev/null || true
sudo systemctl disable telemt 2>/dev/null || true

sudo rm -f /etc/systemd/system/mtpanel.service
sudo rm -rf /etc/systemd/system/mtpanel.service.d
sudo rm -f /etc/systemd/system/telemt.service
sudo systemctl daemon-reload

sudo rm -rf /opt/mtpanel /opt/telemt
sudo rm -rf /etc/mtpanel /etc/telemt
sudo rm -rf /var/lib/mtpanel

sudo userdel mtpanel 2>/dev/null || true
```

## Диагностика

```bash
sudo systemctl status mtpanel --no-pager -l
sudo journalctl -u mtpanel -n 120 --no-pager -l
curl -I http://127.0.0.1:8080/
```

---

- Репозиторий: <https://github.com/NikitaKHS/mtpanel>
- English: [README.en.md](README.en.md)
