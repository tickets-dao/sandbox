# Получение имени текущей папки
CURRENT_DIR := $(notdir $(shell pwd))

# Получение последнего компонента имени папки
NUMBER := $(patsubst tickets%,%,$(CURRENT_DIR))

clone:
	echo "Number is $(NUMBER)"
	mkdir "../tickets$(shell expr $(NUMBER) + 1)"
	cp -v .prepare Makefile "../tickets$(shell expr $(NUMBER) + 1)"
	mv "tickets$(NUMBER)" "../tickets$(shell expr $(NUMBER) + 1)/tickets$(shell expr $(NUMBER) + 1)"

