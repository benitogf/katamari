-- --------------------------------------------------------
-- Host:                         127.0.0.1
-- Server version:               10.3.12-MariaDB - mariadb.org binary distribution
-- Server OS:                    Win64
-- HeidiSQL Version:             9.4.0.5125
-- --------------------------------------------------------

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET NAMES utf8 */;
/*!50503 SET NAMES utf8mb4 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;


-- Dumping database structure for samo
CREATE DATABASE IF NOT EXISTS `samo` /*!40100 DEFAULT CHARACTER SET utf8 */;
USE `samo`;

-- Dumping structure for procedure samo.del
DELIMITER //
CREATE DEFINER=`root`@`localhost` PROCEDURE `del`(
	IN `key` VARCHAR(255)
)
BEGIN
	DELETE FROM `keys` WHERE `keys`.`key` = `key`;
END//
DELIMITER ;

-- Dumping structure for procedure samo.getMo
DELIMITER //
CREATE DEFINER=`root`@`localhost` PROCEDURE `getMo`(
	IN `key` VARCHAR(255)
)
BEGIN
   SET `key` = CONCAT(`key`, '/');
	SELECT TRIM(LEADING `key` from `keys`.`key`) AS 'Index', `values`.`data` AS 'Data', `keys`.created AS 'Created', `keys`.updated AS 'Updated'
	FROM `keys`
	JOIN `values` ON `values`.`key_id` = `keys`.`id`
	WHERE `keys`.`key` LIKE CONCAT(`key`,'%')
	AND TRIM(LEADING `key` from `keys`.`key`) NOT LIKE '%/%';
END//
DELIMITER ;

-- Dumping structure for procedure samo.getSa
DELIMITER //
CREATE DEFINER=`root`@`localhost` PROCEDURE `getSa`(
	IN `key` VARCHAR(255)
)
BEGIN
	SELECT `keys`.`key` AS 'Index', `values`.`data` AS 'Data', `keys`.created AS 'Created', `keys`.updated AS 'Updated'
	FROM `keys`
	JOIN `values` ON `values`.`key_id` = `keys`.`id`
	WHERE `keys`.`key` = `key`;
END//
DELIMITER ;

-- Dumping structure for procedure samo.keys
DELIMITER //
CREATE DEFINER=`root`@`localhost` PROCEDURE `keys`()
BEGIN
  SELECT `keys`.`key` from `keys`;
END//
DELIMITER ;

-- Dumping structure for table samo.keys
CREATE TABLE IF NOT EXISTS `keys` (
  `id` int(10) NOT NULL AUTO_INCREMENT,
  `key` varchar(255) NOT NULL,
  `created` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  `updated` timestamp NOT NULL DEFAULT '0000-00-00 00:00:00',
  PRIMARY KEY (`id`),
  UNIQUE KEY `pk_unique_key` (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Data exporting was unselected.
-- Dumping structure for procedure samo.set
DELIMITER //
CREATE DEFINER=`root`@`localhost` PROCEDURE `set`(
	IN `key` VARCHAR(255),
	IN `data` VARCHAR(10000)
)
BEGIN
   DECLARE errno INT;
   DECLARE keyID INT;
   DECLARE EXIT HANDLER FOR SQLEXCEPTION
   BEGIN
		GET CURRENT DIAGNOSTICS CONDITION 1 errno = MYSQL_ERRNO;
	   SELECT errno AS MYSQL_ERROR;
	   ROLLBACK;
   END;
	START TRANSACTION;
		INSERT INTO `keys` VALUES(DEFAULT, `key`, now(), '0000-00-00 00:00:00') ON DUPLICATE KEY UPDATE `id`=LAST_INSERT_ID(`id`), `updated`=now();
		SET keyID = LAST_INSERT_ID();
		INSERT INTO `values` VALUES(DEFAULT, keyID, `data`) ON DUPLICATE KEY UPDATE `data`=`data`;
	COMMIT;
END//
DELIMITER ;

-- Dumping structure for table samo.values
CREATE TABLE IF NOT EXISTS `values` (
  `id` int(10) NOT NULL AUTO_INCREMENT,
  `key_id` int(10) NOT NULL,
  `data` varchar(10000) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `pk_unique_key` (`key_id`),
  CONSTRAINT `fk_values_iv` FOREIGN KEY (`key_id`) REFERENCES `keys` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Data exporting was unselected.
/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IF(@OLD_FOREIGN_KEY_CHECKS IS NULL, 1, @OLD_FOREIGN_KEY_CHECKS) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
